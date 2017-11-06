package main

import (
	"fmt"
	"go/build"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/benhinchley/cmd"
	"github.com/benhinchley/pit"

	git "gopkg.in/src-d/go-git.v4"
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
	if err := p.Run(func(env cmd.Environment, c cmd.Command, args []string) error {
		stdout, stderr := env.GetLoggers()
		wd := env.WorkingDir()

		//TODO: search for `.git` dir in wd, then search upwards until we hit $GOPATH

		r, err := git.PlainOpen(wd)
		if err != nil {
			return fmt.Errorf("%s: unable to open repository: %v", c.Name(), err)
		}

		p, _ := filepath.Rel(filepath.Join(build.Default.GOPATH, "src"), wd)

		pkgs, err := pit.Packages()
		if err != nil {
			return fmt.Errorf("%s: unable to list packages: %v", c.Name(), err)
		}

		rstdout, rstderr := env.GetStdio()
		ctx := &context{
			WorkingDir: wd,
			WDPackage:  p,
			Repository: r,
			Packages:   pkgs,
			RawStdout:  rstdout,
			RawStderr:  rstderr,
			out:        stdout,
			err:        stderr,
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
	WorkingDir           string
	WDPackage            string
	Repository           *git.Repository
	Packages             []pit.Package
	RawStdout, RawStderr io.Writer

	out, err *log.Logger
}

func (c *context) Stdout() *log.Logger { return c.out }
func (c *context) Stderr() *log.Logger { return c.err }
