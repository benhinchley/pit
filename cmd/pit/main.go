package main

import (
	"fmt"
	"log"
	"os"

	"github.com/benhinchley/cmd"
	"github.com/intel/tfortools"
)

func main() {
	p, err := cmd.NewProgram("pit", "smartish wrapper around go test", &pitCommand{}, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := p.ParseArgs(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := p.Run(func(env *cmd.Environment, c cmd.Command, args []string) error {
		stdout, stderr := env.GetLoggers()

		cfg := tfortools.NewConfig(tfortools.OptAllFns)

		ctx := &context{
			WorkingDir:     env.WorkingDir,
			TemplateConfig: cfg,
			out:            stdout,
			err:            stderr,
		}
		if err := c.Run(ctx, args); err != nil {
			return fmt.Errorf("%s: %v", c.Name(), err)
		}
		return nil
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type context struct {
	WorkingDir     string
	TemplateConfig *tfortools.Config

	out, err *log.Logger
}

func (c *context) Stdout() *log.Logger { return c.out }
func (c *context) Stderr() *log.Logger { return c.err }
