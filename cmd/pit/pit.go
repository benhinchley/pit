package main

import (
	"flag"
	"fmt"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/benhinchley/cmd"

	git "gopkg.in/src-d/go-git.v4"
)

type pitCommand struct{}

func (cmd *pitCommand) Name() string           { return "pit" }
func (cmd *pitCommand) Args() string           { return "" }
func (cmd *pitCommand) Desc() string           { return "smartish wrapper around go test" }
func (cmd *pitCommand) Help() string           { return "TODO" }
func (cmd *pitCommand) Register(*flag.FlagSet) {}

func (cmd *pitCommand) Run(ctx cmd.Context, args []string) error {
	cf, err := changedFiles(ctx.(*context).WDPackage, ctx.(*context).Repository)
	if err != nil {
		return fmt.Errorf("%s: %v", cmd.Name(), err)
	}

	if len(cf) == 0 {
		ctx.Stderr().Println("no changes")
		return nil
	}

	cp := changedPackages(cf)

	// TODO: Setup some sort of work pool for this
	for _, chpkg := range cp {
		for _, pkg := range ctx.(*context).Packages {
			if pkg.Name == chpkg {
				tests, err := pkg.Tests()
				if err != nil {
					ctx.Stderr().Printf("unable to get test files for %s: %v\n", pkg.Name, err)
					continue
				}

				if len(tests) == 0 {
					ctx.Stderr().Printf("no tests for %s\n", pkg.Name)
					continue
				}
				gtc := exec.Command("go", "test", "-v", "-cover", pkg.Name)
				gtc.Stdout = ctx.(*context).RawStdout
				gtc.Stderr = ctx.(*context).RawStderr

				if err := gtc.Run(); err != nil {
					ctx.Stderr().Printf("unable to run \"go test -v -cover %s\": %v\n", pkg.Name, err)
				}
			}
		}
	}

	return nil
}

func changedFiles(p string, r *git.Repository) ([]string, error) {
	f := []string{}
	w, err := r.Worktree()
	if err != nil {
		return f, fmt.Errorf("unable to get repository worktree: %s", err)
	}
	s, err := w.Status()
	if err != nil {
		return f, fmt.Errorf("unable to worktree status: %s", err)
	}

	for file, status := range s {
		if status.Worktree != git.Unmodified {
			f = append(f, filepath.Join(p, strings.TrimRight(file, ",")))
		}
	}
	f = filter(f, func(s string) bool {
		return path.Ext(s) == ".go"
	})

	return f, nil
}

func changedPackages(f []string) []string {
	r := []string{}
	for _, file := range f {
		r = append(r, filepath.Dir(file))
	}
	return removeDuplicates(r)
}

func filter(vs []string, f func(string) bool) []string {
	r := make([]string, 0)
	for _, v := range vs {
		if f(v) {
			r = append(r, v)
		}
	}
	return r
}

func removeDuplicates(d []string) []string {
	s := map[string]bool{}
	r := []string{}

	for v := range d {
		if s[d[v]] != true {
			s[d[v]] = true
			r = append(r, d[v])
		}
	}
	return r
}
