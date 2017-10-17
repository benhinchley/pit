package main

import (
	"fmt"
	"go/build"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	git "gopkg.in/src-d/go-git.v4"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to get working directory: %s\n", err)
		os.Exit(1)
	}

	r, err := git.PlainOpen(wd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to open repository: %s\n", err)
		os.Exit(1)
	}

	p, _ := filepath.Rel(filepath.Join(build.Default.GOPATH, "src"), wd)

	pkgs, err := Packages()
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to list packages: %s\n", err)
		os.Exit(1)
	}

	c := &Config{
		WorkingDir:        wd,
		WorkingDirPackage: p,
		Repository:        r,
		Packages:          pkgs,
	}

	os.Exit(c.Run(os.Args[1:]))
}

type Config struct {
	WorkingDir        string
	WorkingDirPackage string
	Repository        *git.Repository
	Packages          []Package
}

func (c *Config) Run(args []string) int {
	// ref, _ := c.Repository.Head() // ref.Hash().String()[:7]
	cf, err := changedFiles(c.WorkingDirPackage, c.Repository)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return 1
	}

	if len(cf) == 0 {
		fmt.Fprintf(os.Stderr, "no changes")
		return 0
	}

	cp := changedPackages(cf)

	// TODO: Setup some sort of work pool for this
	for _, chpkg := range cp {
		for _, pkg := range c.Packages {
			if pkg.Name == chpkg {
				tests, err := pkg.Tests()
				if err != nil {
					fmt.Fprintf(os.Stderr, "unable to get test files for %s: %s", pkg.Name, err)
					continue
				}

				if len(tests) == 0 {
					fmt.Fprintf(os.Stderr, "no tests for %s\n", pkg.Name)
					continue
				}
				cmd := exec.Command("go", "test", "-v", "-cover", pkg.Name)
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr

				if err := cmd.Run(); err != nil {
					fmt.Fprintf(os.Stderr, "unable to run \"go test -v -cover %s\": %s", pkg.Name, err)
				}
			}
		}
	}

	return 0
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
	f = Filter(f, func(s string) bool {
		return path.Ext(s) == ".go"
	})

	return f, nil
}

func changedPackages(f []string) []string {
	r := []string{}
	for _, file := range f {
		r = append(r, filepath.Dir(file))
	}
	return RemoveDuplicates(r)
}

func Filter(vs []string, f func(string) bool) []string {
	r := make([]string, 0)
	for _, v := range vs {
		if f(v) {
			r = append(r, v)
		}
	}
	return r
}

func RemoveDuplicates(d []string) []string {
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
