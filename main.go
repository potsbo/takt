package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kvz/logstreamer"
	"github.com/pkg/errors"
	"github.com/potsbo/takt/pkg/task"
	"golang.org/x/sync/errgroup"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	path := ".takt.yaml"
	file, err := os.Open(path)
	if err != nil {
		return errors.Wrap(err, "failed to open file")
	}
	defer file.Close()

	takt, err := task.Parse(file)
	if err != nil {
		return err
	}

	tasks := task.FromTakt(*takt)

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
