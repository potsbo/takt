package task

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/izumin5210/execx"
)

var (
	DependencyNotFulfilledErr = errors.New("Dependency not filled")
)

type Task struct {
	Name             string
	Command          string
	Depends          []string
	doneNotification []chan TaskNotification
	waiting          []chan TaskNotification
}

type TaskNotification struct {
	ok   bool
	name string
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
	cmd := execx.CommandContext(ctx, "sh", "-c", t.Command)
	cmd.Stdout = prefixLogger
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	if err := cmd.Wait(); err != nil {
		return err
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
