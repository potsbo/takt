package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kvz/logstreamer"
	"github.com/potsbo/takt/pkg/task"
	"golang.org/x/sync/errgroup"
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
			Command: "echo task2; sleep 2; echoaeuaoeu done",
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

			if err := t.Run(ctx, prefixLogger); err != nil {
				cancel()
				return err
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}
