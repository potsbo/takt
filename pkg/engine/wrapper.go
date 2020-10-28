package engine

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/izumin5210/execx"
	"github.com/pkg/errors"
)

type wrappedRunner struct {
	original Runner
}

func WrapInterrupt(r Runner) Runner {
	return &wrappedRunner{
		original: r,
	}
}

func (r wrappedRunner) Run(ctx context.Context, opts ...Option) error {
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

	err := r.original.Run(ctx, opts...)
	close(cancelReceived)

	if es, ok := err.(*execx.ExitStatus); ok {
		if es.Signaled {
			return nil
		}
	}
	return errors.Wrap(err, "runner exited with error")
}
