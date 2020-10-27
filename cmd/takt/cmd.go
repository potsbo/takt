package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/izumin5210/clig/pkg/clib"
	"github.com/izumin5210/execx"
	"github.com/kvz/logstreamer"
	"github.com/pkg/errors"
	"github.com/potsbo/takt/pkg/task"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

type runFunc func(ctx context.Context) error

func newCmd(ioset *clib.IO) *cobra.Command {
	runner := taktRunner{
		ioset:        ioset,
		taktfilePath: ".takt.yaml",
	}

	cmd := &cobra.Command{
		Use:   "takt",
		Short: "Task runner with cancel",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			return wrapInterrupt(runner.Run)(ctx)
		},
	}

	cmd.Flags().StringVarP(&runner.taktfilePath, "file", "f", runner.taktfilePath, "path to TaktFile")

	cmd.SetOut(ioset.Out)
	cmd.SetErr(ioset.Err)
	cmd.SetIn(ioset.In)

	return cmd
}

func wrapInterrupt(runner runFunc) runFunc {
	return func(ctx context.Context) error {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		cancelReceived := make(chan os.Signal, 1)
		signal.Notify(cancelReceived, os.Interrupt, syscall.SIGTERM, os.Kill)
		clean := func() {
			<-cancelReceived
			fmt.Println("Interrupted, cleaning up...")
			cancel()
		}
		go clean()

		err := runner(ctx)
		close(cancelReceived)

		if es, ok := err.(*execx.ExitStatus); ok {
			if es.Signaled {
				return nil
			}
		}
		return errors.Wrap(err, "runner exited with error")
	}
}

type taktRunner struct {
	taktfilePath string
	ioset        *clib.IO
}

func (t taktRunner) Run(ctx context.Context) error {
	file, err := os.Open(t.taktfilePath)
	if err != nil {
		return errors.Wrap(err, "failed to open file")
	}
	defer file.Close()

	takt, err := task.Parse(file)
	if err != nil {
		return err
	}

	tasks, err := task.FromTakt(*takt)
	if err != nil {
		return err
	}

	eg, ctx := errgroup.WithContext(ctx)

	logger := log.New(t.ioset.Out, "", log.Ldate|log.Ltime)
	for _, t := range tasks {
		t := t
		eg.Go(func() error {
			prefixLogger := logstreamer.NewLogstreamer(logger, fmt.Sprintf("%s: ", t.Name), false)
			defer prefixLogger.Close()

			return t.Run(ctx, prefixLogger)
		})
	}

	return eg.Wait()
}
