package task

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/izumin5210/execx"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

var (
	DependencyNotFulfilledErr = errors.New("Dependency not filled")
)

type Takt struct {
	Tasks map[string]Task
}

type Task struct {
	Name             string
	Steps            []Step
	Depends          []string
	Tags             []string
	Env              map[string]string
	doneNotification []chan TaskNotification
	waiting          []chan TaskNotification
}

type Step struct {
	Run string
}

type TaskNotification struct {
	ok   bool
	name string
}

func FromTakt(takt Takt) []Task {
	ts := []Task{}
	for name, tsk := range takt.Tasks {
		tsk := tsk
		tsk.Name = name
		ts = append(ts, tsk)
	}
	return ts
}

func (t *Task) DependsOn(dependedTask *Task) {
	c := make(chan TaskNotification)
	dependedTask.doneNotification = append(dependedTask.doneNotification, c)
	t.waiting = append(t.waiting, c)
}

func (t Task) waitDependecies() error {
	okToGo := true
	for _, c := range t.waiting {
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

func (t Task) execute(ctx context.Context, prefixLogger io.Writer) error {
	fmt.Fprintf(prefixLogger, "starting\n")
	for _, step := range t.Steps {
		cmd := execx.CommandContext(ctx, "sh", "-c", step.Run)
		cmd.Stdout = prefixLogger
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			return err
		}
		if err := cmd.Wait(); err != nil {
			return err
		}
	}
	return nil
}

func (t Task) Run(ctx context.Context, prefixLogger io.Writer) error {
	runner := func() error {
		if err := t.waitDependecies(); err != nil {
			return err
		}
		if err := t.execute(ctx, prefixLogger); err != nil {
			return err
		}
		return nil
	}

	if err := runner(); err != nil {
		for _, done := range t.doneNotification {
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
	for _, done := range t.doneNotification {
		done <- TaskNotification{
			ok:   true,
			name: t.Name,
		}
	}

	return nil
}

func Parse(file io.Reader) (*Takt, error) {
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read file")
	}
	root := Takt{}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&root); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal yaml")
	}

	return &root, nil
}
