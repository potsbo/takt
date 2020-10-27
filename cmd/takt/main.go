package main

import (
	"fmt"
	"os"

	"github.com/izumin5210/clig/pkg/clib"
)

func main() {
	defer clib.Close()

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	return newCmd(clib.Stdio()).Execute()
}
