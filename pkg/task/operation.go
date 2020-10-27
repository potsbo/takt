package task

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
)

var (
	DependencyNotFulfilledErr = errors.New("Dependency not filled")
)

type Operation struct {
	Name   string
	task   *Task
	status status
}

type status struct {
	doneNotification []chan TaskNotification
	waiting          []chan TaskNotification
}

func FromTakt(takt Takt) ([]*Operation, error) {
	ts := []*Operation{}
	for name, tsk := range takt.Tasks {
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
	c := make(chan TaskNotification)
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
		return DependencyNotFulfilledErr
	}

	return nil
}

func (t Operation) Run(ctx context.Context, prefixLogger io.Writer) error {
	runner := func() error {
		if err := t.waitDependecies(); err != nil {
			return err
		}
		if err := t.task.execute(ctx, prefixLogger); err != nil {
			return err
		}
		return nil
	}

	if err := runner(); err != nil {
		for _, done := range t.status.doneNotification {
			done <- TaskNotification{
				ok:   false,
				name: t.Name,
			}
		}
		if err == DependencyNotFulfilledErr {
			return nil
		}
		fmt.Fprintf(prefixLogger, "runner finished with err, %v\n", err)
		return err
	}

	fmt.Fprintf(prefixLogger, "done\n")
	for _, done := range t.status.doneNotification {
		done <- TaskNotification{
			ok:   true,
			name: t.Name,
		}
	}

	return nil
}
