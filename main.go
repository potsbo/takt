package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/izumin5210/execx"
	"github.com/kvz/logstreamer"
	"github.com/potsbo/takt/pkg/task"
	"golang.org/x/sync/errgroup"
)

const (
	debug = false
)

var (
	DependencyNotFulfilledErr = errors.New("Dependency not filled")
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	tasks := []task.Task{
		{
			Name:    "task1",
			Command: "echo task1; sleep 2; echo done",
			Depends: nil,
		},
		{
			Name:    "task2",
			Command: "echo task2; sleep 2; echo done",
			Depends: []string{"task1"},
		},
		{
			Name:    "task3",
			Command: "echo task3",
			Depends: []string{"task1"},
		},
		{
			Name:    "task4",
			Command: "while true; do echo test; sleep 1s; done",
			Depends: []string{"task2", "task3"},
		},
		{
			Name:    "task5",
			Command: "while true; do echo test; sleep 1s; done",
			Depends: []string{"task2", "task3"},
		},
	}

	taskMap := map[string]*task.Task{}
	for _, tsk := range tasks {
		tsk := tsk
		taskMap[tsk.Name] = &tsk
	}

	for _, t := range taskMap {
		for _, dependedTaskName := range t.Depends {
			dependedTask, ok := taskMap[dependedTaskName]
			if !ok {
				return errors.New("task not found")
			}
			t.DependsOn(dependedTask)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()

	eg := errgroup.Group{}

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)
	for _, t := range taskMap {

		t := t
		eg.Go(func() error {
			prefixLogger := logstreamer.NewLogstreamer(logger, fmt.Sprintf("%s: ", t.Name), false)
			defer prefixLogger.Close()

			runner := func() error {
				okToGo := true
				for _, c := range t.Waiting {
					notification := <-c
					if debug {
						prefixLogger.Logger.Println(t.Name, notification)
					}
					if !notification.Ok {
						okToGo = notification.Ok
					}
				}
				if !okToGo {
					return DependencyNotFulfilledErr
				}
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

			if err := runner(); err != nil {
				cancel()
				for _, done := range t.DoneNotification {
					done <- task.TaskNotification{
						Ok:   false,
						Name: t.Name,
					}
				}
				if err == DependencyNotFulfilledErr {
					return nil
				}
				fmt.Fprintf(prefixLogger, "runner finished with err, %v\n", err)
				return err
			}

			fmt.Fprintf(prefixLogger, "done\n")
			for _, done := range t.DoneNotification {
				done <- task.TaskNotification{
					Ok:   true,
					Name: t.Name,
				}
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}
