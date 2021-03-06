package operation

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/potsbo/takt/pkg/task"
)

var (
	dependencyNotFulfilledErr = errors.New("Dependency not filled")
)

type Operation struct {
	Name   string
	task   *task.Task
	status status
}

type status struct {
	doneNotification []chan taskNotification
	waiting          []chan taskNotification
}

type taskNotification struct {
	ok   bool
	name string
}

type option struct {
	tagOnlyList []string
}

func (o option) available(t task.Task) bool {
	if len(o.tagOnlyList) > 0 {
		//  TODO: better logic
		for _, only := range o.tagOnlyList {
			for _, tag := range t.Tags {
				if only == tag {
					return true
				}
			}
		}
		return false
	}

	return true
}

type Option func(*option)

var nopOption = func(*option) {}

func WithOnlyTags(tags ...string) Option {
	if len(tags) == 0 {
		return nopOption
	}
	return func(opt *option) {
		opt.tagOnlyList = tags
	}
}

func FromTakt(takt task.Takt, opts ...Option) ([]*Operation, error) {
	opt := option{}
	for _, op := range opts {
		op(&opt)
	}

	ts := []*Operation{}
	for name, tsk := range takt.Tasks {
		if !opt.available(tsk) {
			continue
		}
		tsk := tsk
		op := Operation{
			Name:   name,
			task:   &tsk,
			status: status{},
		}
		ts = append(ts, &op)
	}

	if err := resolveDeps(ts); err != nil {
		return nil, errors.Wrap(err, "failed to resolve dependencies")
	}

	return ts, nil
}

func resolveDeps(operations []*Operation) error {
	opMap := map[string]*Operation{}
	for _, op := range operations {
		opMap[op.Name] = op
	}

	for _, t := range opMap {
		for _, dependedTaskName := range t.task.Depends {
			dependedTask, ok := opMap[dependedTaskName]
			if !ok {
				return errors.New("task not found")
			}
			t.dependsOn(dependedTask)
		}
	}

	return nil
}

func (t *Operation) dependsOn(dependedTask *Operation) {
	c := make(chan taskNotification)
	dependedTask.status.doneNotification = append(dependedTask.status.doneNotification, c)
	t.status.waiting = append(t.status.waiting, c)
}

func (t Operation) waitDependecies() error {
	okToGo := true
	fmt.Printf("%s waits %d tasks\n", t.Name, len(t.status.waiting))
	for _, c := range t.status.waiting {
		notification := <-c
		if !notification.ok {
			okToGo = notification.ok
		}
	}
	if !okToGo {
		return dependencyNotFulfilledErr
	}

	return nil
}

func (t Operation) Run(ctx context.Context, logger io.Writer) error {
	runner := func() error {
		if err := t.waitDependecies(); err != nil {
			return err
		}
		if err := t.task.Execute(ctx, logger); err != nil {
			return err
		}
		return nil
	}

	if err := runner(); err != nil {
		for _, done := range t.status.doneNotification {
			done <- taskNotification{
				ok:   false,
				name: t.Name,
			}
		}
		if err == dependencyNotFulfilledErr {
			return nil
		}
		fmt.Fprintf(logger, "runner finished with err, %v\n", err)
		return err
	}

	fmt.Fprintf(logger, "done\n")
	for _, done := range t.status.doneNotification {
		done <- taskNotification{
			ok:   true,
			name: t.Name,
		}
	}

	return nil
}
