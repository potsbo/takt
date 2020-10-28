package engine

import (
	"context"
	"hash/fnv"
	"log"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/izumin5210/clig/pkg/clib"
	"github.com/kvz/logstreamer"
	"github.com/pkg/errors"
	"github.com/potsbo/takt/pkg/operation"
	"github.com/potsbo/takt/pkg/task"
	"golang.org/x/sync/errgroup"
)

type Runner interface {
	Run(ctx context.Context, opts ...Option) error
}

type Option func(*option)

var nopOption = func(*option) {}

type option struct {
	taktfilePath string
	tagOnlyList  []string
}

func New(ioset *clib.IO) Runner {
	return runner{
		ioset: ioset,
	}
}

type runner struct {
	ioset *clib.IO
}

func (t runner) Run(ctx context.Context, opts ...Option) error {
	opt := option{}
	for _, op := range opts {
		op(&opt)
	}

	taktfilePath := opt.taktfilePath

	if taktfilePath == "" {
		return errors.New("No Takt file path configured")
	}

	file, err := os.Open(taktfilePath)
	if err != nil {
		return errors.Wrap(err, "failed to open file")
	}
	defer file.Close()

	takt, err := task.Parse(file)
	if err != nil {
		return err
	}

	tasks, err := operation.FromTakt(*takt)
	if err != nil {
		return err
	}

	eg, ctx := errgroup.WithContext(ctx)

	max := 0
	for _, tsk := range tasks {
		if len(tsk.Name) > max {
			max = len(tsk.Name)
		}
	}

	logger := log.New(t.ioset.Out, "", log.Ldate|log.Ltime)
	for _, t := range tasks {
		t := t
		eg.Go(func() error {
			spaces := strings.Repeat(" ", max-len(t.Name)+1)
			c := determineColor(t.Name)
			prefixLogger := logstreamer.NewLogstreamer(logger, c.Sprint(t.Name+spaces), false)
			defer prefixLogger.Close()

			return t.Run(ctx, prefixLogger)
		})
	}

	return eg.Wait()
}

func WithOnlyTags(tags ...string) Option {
	if len(tags) == 0 {
		return nopOption
	}
	return func(opt *option) {
		opt.tagOnlyList = tags
	}
}

var colorList = []*color.Color{
	color.New(color.FgHiCyan),
	color.New(color.FgHiGreen),
	color.New(color.FgHiMagenta),
	color.New(color.FgHiYellow),
	color.New(color.FgHiBlue),
	color.New(color.FgHiRed),
}

func determineColor(key string) *color.Color {
	hash := fnv.New32()
	hash.Write([]byte(key))
	idx := hash.Sum32() % uint32(len(colorList))

	return colorList[idx]
}
