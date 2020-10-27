package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/izumin5210/clig/pkg/clib"
	"github.com/kvz/logstreamer"
	"github.com/pkg/errors"
	"github.com/potsbo/takt/pkg/task"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

func newCmd(ioset *clib.IO) *cobra.Command {
	taktfilePath := ".takt.yaml"
	cmd := &cobra.Command{
		Use:   "takt",
		Short: "Task runner with cancel",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			runner := func(ctx context.Context) error {
				file, err := os.Open(taktfilePath)
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

				eg := errgroup.Group{}

				logger := log.New(ioset.Out, "", log.Ldate|log.Ltime)
				for _, t := range tasks {
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

			{
				done := make(chan bool, 1)

				c := make(chan os.Signal, 1)
				signal.Notify(c, os.Interrupt, syscall.SIGTERM, os.Kill)
				go func() {
					<-c
					fmt.Println("Interrupted, cleaning up...")
					cancel()
					done <- true
				}()

				err := runner(ctx)
				close(c)
				<-done
				return errors.Wrap(err, "failed to run")
			}
		},
	}

	cmd.Flags().StringVarP(&taktfilePath, "file", "f", taktfilePath, "path to TaktFile")
	cmd.SetOut(ioset.Out)
	cmd.SetErr(ioset.Err)
	cmd.SetIn(ioset.In)

	return cmd
}
