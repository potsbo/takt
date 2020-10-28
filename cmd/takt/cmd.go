package main

import (
	"context"

	"github.com/izumin5210/clig/pkg/clib"
	"github.com/octago/sflags/gen/gpflag"
	"github.com/pkg/errors"
	"github.com/potsbo/takt/pkg/engine"
	"github.com/spf13/cobra"
)

type rootOption struct {
	Taktfile string
	Only     []string
}

func newCmd(ioset *clib.IO) *cobra.Command {
	opts := &rootOption{
		Taktfile: ".takt.yaml",
	}

	cmd := &cobra.Command{
		Use:   "takt",
		Short: "Task runner with cancel",
		RunE: func(cmd *cobra.Command, args []string) error {
			runner := engine.New(ioset)
			ctx := context.Background()

			options := []engine.Option{
				engine.WithOnlyTags(opts.Only...),
			}

			return engine.WrapInterrupt(runner).Run(ctx, options...)
		},
		SilenceUsage: true,
	}

	if err := gpflag.ParseTo(opts, cmd.Flags()); err != nil {
		panic(errors.Wrap(err, "failed to setup flags for root command"))
	}

	cmd.SetOut(ioset.Out)
	cmd.SetErr(ioset.Err)
	cmd.SetIn(ioset.In)

	return cmd
}
