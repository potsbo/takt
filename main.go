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
	"golang.org/x/sync/errgroup"
)

const (
	debug = false
)

var (
	DependencyNotFulfilledErr = errors.New("Dependency not filled")
)

type Task struct {
	name             string
	command          string
	depends          []string
	doneNotification []chan TaskNotification
	waiting          []chan TaskNotification
}

type TaskNotification struct {
	ok   bool
	name string
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	tasks := []Task{
		{
			name:    "task1",
			command: "echo task1; sleep 2; echo done",
			depends: nil,
		},
		{
			name:    "task2",
			command: "echo task2; sleep 2; echo done",
			depends: []string{"task1"},
		},
		{
			name:    "task3",
			command: "echo task3",
			depends: []string{"task1"},
		},
		{
			name:    "task4",
			command: "while true; do echo test; sleep 1s; done",
			depends: []string{"task2", "task3"},
		},
		{
			name:    "task5",
			command: "while true; do echo test; sleep 1s; done",
			depends: []string{"task2", "task3"},
		},
	}

	taskMap := map[string]Task{}
	for _, task := range tasks {
		taskMap[task.name] = task
	}

	for _, task := range taskMap {
		for _, dependedTaskName := range task.depends {
			dependedTask, ok := taskMap[dependedTaskName]
			if !ok {
				return errors.New("task not found")
			}
			c := make(chan TaskNotification)
			dependedTask.doneNotification = append(dependedTask.doneNotification, c)
			taskMap[dependedTaskName] = dependedTask
			task.waiting = append(task.waiting, c)
			taskMap[task.name] = task
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()

	eg := errgroup.Group{}

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)
	for _, task := range taskMap {

		task := task
		eg.Go(func() error {
			prefixLogger := logstreamer.NewLogstreamer(logger, fmt.Sprintf("%s: ", task.name), false)
			defer prefixLogger.Close()

			runner := func() error {
				okToGo := true
				for _, c := range task.waiting {
					notification := <-c
					if debug {
						prefixLogger.Logger.Println(task.name, notification)
					}
					if !notification.ok {
						okToGo = notification.ok
					}
				}
				if !okToGo {
					return DependencyNotFulfilledErr
				}
				fmt.Fprintf(prefixLogger, "starting\n")
				cmd := execx.CommandContext(ctx, "sh", "-c", task.command)
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
				for _, done := range task.doneNotification {
					done <- TaskNotification{
						ok:   false,
						name: task.name,
					}
				}
				if err == DependencyNotFulfilledErr {
					return nil
				}
				fmt.Fprintf(prefixLogger, "runner finished with err, %v\n", err)
				return err
			}

			fmt.Fprintf(prefixLogger, "done\n")
			for _, done := range task.doneNotification {
				done <- TaskNotification{
					ok:   true,
					name: task.name,
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
