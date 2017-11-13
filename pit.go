// Package pit ...
package pit

import (
	"bytes"
	"fmt"
	"go/build"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	git "gopkg.in/src-d/go-git.v4"

	"github.com/benhinchley/pit/internals/testparser"
)

// FindPackages returns a slice of Packages found in the passed directory.
func FindPackages(wd string) ([]*Package, error) {
	ctx := build.Default

	var res []*Package
	for dir := range findDir(wd) {
		pkg, err := ctx.ImportDir(dir, build.IgnoreVendor)
		if err != nil {
			if _, ok := err.(*build.NoGoError); ok {
				continue
			}
			return nil, err
		}
		res = append(res, &Package{
			Name:        pkg.Name,
			Dir:         pkg.Dir,
			ImportPath:  pkg.ImportPath,
			SourceFiles: pkg.GoFiles,
			TestFiles:   pkg.TestGoFiles,
		})
	}
	return res, nil
}

func findDir(p string) <-chan string {
	w := make(fileWalk)
	go func() {
		if err := filepath.Walk(p, w.Walk); err != nil {
			log.Printf("error: %s", err)
		}
		close(w)
	}()
	return w
}

type fileWalk chan string

func (f fileWalk) Walk(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if info.IsDir() {
		if !strings.Contains(path, "vendor") && !strings.Contains(path, ".git") {
			f <- path
		}
	}
	return nil
}

type Package struct {
	Name        string
	Dir         string
	ImportPath  string
	SourceFiles []string
	TestFiles   []string

	// Some of these are a little time consuming to create
	repo     *git.Repository
	worktree *git.Worktree
	status   map[string]*git.FileStatus
}

// Repository returns the git repository for the package
func (p *Package) Repository() (*git.Repository, error) {
	if p.repo == nil {
		if exists(filepath.Join(p.Dir, ".git")) {
			r, err := git.PlainOpen(p.Dir)
			if err != nil {
				return nil, fmt.Errorf("unable to open repository: %v", err)
			}
			p.repo = r
			return r, nil
		}

		// Search upwards until we find the git toplevel
		// git rev-parse --show-toplevel
		// NOTE: This is not the smartest impl as if you are not in a git dir
		// It will just block here until it errors out saying it can't
		// any higher than /
		path, _ := filepath.Rel(p.Dir, filepath.Join(p.Dir, ".git"))
		for {
			if exists(path) {
				r, err := git.PlainOpen(filepath.Dir(path))
				if err != nil {
					return nil, fmt.Errorf("unable to open repository: %v", err)
				}
				p.repo = r
				return r, nil
			}
			path = filepath.Join("../", path)
		}
	}

	return p.repo, nil
}

func exists(p string) bool {
	_, err := os.Stat(p)
	if os.IsNotExist(err) {
		return false
	}
	return true
}

// RunTests runs the packages test suite, checking if the package has tests
// and whether any files have been changed
func (p *Package) RunTests() (*testparser.PackageResult, error) {
	ok, err := p.hasChangedFiles()
	if err != nil {
		return nil, err
	}
	if !ok {
		return &testparser.PackageResult{
			Name:    p.ImportPath,
			Status:  testparser.StatusSkip,
			Summary: "[no changed files]",
		}, nil
	}

	if !p.hasTestFiles() {
		return &testparser.PackageResult{
			Name:    p.ImportPath,
			Status:  testparser.StatusSkip,
			Summary: "[no test files]",
		}, nil
	}

	var out bytes.Buffer
	cmd := exec.Command("go", "test", "-v", "-cover", p.ImportPath)
	cmd.Stdout = &out
	cmd.Stderr = &out
	cmd.Run()

	r, err := testparser.Parse(&out)
	if err != nil {
		return nil, fmt.Errorf("unable to parse test output: %v", err)
	}

	return r[0], nil
}

func (p *Package) hasChangedFiles() (bool, error) {
	if p.repo == nil {
		if _, err := p.Repository(); err != nil {
			return false, err
		}
	}

	if p.worktree == nil {
		t, err := p.repo.Worktree()
		if err != nil {
			return false, err
		}
		p.worktree = t
	}

	if p.status == nil {
		s, err := p.worktree.Status()
		if err != nil {
			return false, err
		}
		p.status = s
	}

	// The following block of code seems a bit wasteful
	files := []string{}
	for file, status := range p.status {
		if status.Worktree != git.Unmodified {
			files = append(files, file)
		}
	}

	files = filter(files, func(p string) bool {
		return path.Ext(p) == ".go"
	})

	if len(files) > 0 {
		return true, nil
	}

	return false, nil
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

func (p *Package) hasTestFiles() bool {
	switch len(p.TestFiles) {
	case 0:
		return false
	default:
		return true
	}
}
