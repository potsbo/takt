package task

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/izumin5210/execx"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type Takt struct {
	Tasks map[string]Task
}

type Task struct {
	Steps   []Step
	Depends []string
	Tags    []string
	Env     map[string]string
}

type Step struct {
	Run string
}

func (t Task) Execute(ctx context.Context, prefixLogger io.Writer) error {
	fmt.Fprintf(prefixLogger, "starting\n")
	for _, step := range t.Steps {
		cmd := execx.CommandContext(ctx, "sh", "-c", step.Run)
		cmd.Stdout = prefixLogger
		cmd.Stderr = os.Stderr
		if err := cmd.Start(); err != nil {
			return err
		}
		if err := cmd.Wait(); err != nil {
			return err
		}
	}
	return nil
}

func Parse(file io.Reader) (*Takt, error) {
	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read file")
	}
	root := Takt{}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&root); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal yaml")
	}

	return &root, nil
}
