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
	"regexp"
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
	repoDir  string
	repo     *git.Repository
	worktree *git.Worktree
	status   map[string]*git.FileStatus
}

// Repository returns the git repository for the package
func (p *Package) Repository() (*git.Repository, error) {
	if p.repo == nil {
		if r, err := git.PlainOpen(p.Dir); err == nil {
			p.repoDir = p.Dir
			p.repo = r
			return r, nil
		}

		// search upwards until we find a .git directory.
		path := filepath.Join(p.Dir, "../")
		for {
			if r, err := git.PlainOpen(path); err == nil {
				p.repoDir = path
				p.repo = r
				return r, nil
			}
			path = filepath.Join(path, "../")
		}
	}

	return p.repo, nil
}

// TestConfig defines how a packages tests will be run
type TestConfig struct {
	RunAll     bool
	CommitHash string
	Args       []string
}

// RunTests runs the packages test suite, checking if the package has tests
// and whether any files have been changed
func (p *Package) RunTests(config *TestConfig) (*PackageTestResult, error) {
	if !p.hasTestFiles() {
		return &PackageTestResult{
			Name:    p.ImportPath,
			Status:  testparser.StatusSkip.String(),
			Summary: "[no test files]",
		}, nil
	}

	tests, err := p.determineTestsToRun(config.CommitHash)
	if err != nil {
		return nil, fmt.Errorf("unable to determine tests to run: %v", err)
	}

	// no changed files or tests to run
	if tests == nil {
		return &PackageTestResult{
			Name:    p.ImportPath,
			Status:  testparser.StatusSkip.String(),
			Summary: "[no changed files]",
		}, nil
	}

	goTestArgs := []string{"test", "-v", "-cover", "-run", tests.String()}
	if config.Args != nil {
		goTestArgs = append(goTestArgs, config.Args...)
	}
	goTestArgs = append(goTestArgs, p.ImportPath)

	var out bytes.Buffer
	cmd := exec.Command("go", goTestArgs...)
	cmd.Stdout = &out
	cmd.Stderr = &out
	cmd.Run()

	r, err := testparser.Parse(&out)
	if err != nil {
		return nil, fmt.Errorf("unable to parse test output: %v", err)
	}

	return fromTestparser(r[0]), nil
}

// determineTestsToRun returns a regexp that can be passed to
// the `-run` flag of `go test`.
func (p *Package) determineTestsToRun(hash string) (*regexp.Regexp, error) {
	if hash == "" {
		dirty, _, err := p.hasDirtyWorktree()
		if err != nil {
			return nil, fmt.Errorf("unable to determine worktree state: %v", err)
		}
		if !dirty {
			return nil, nil
		}

		return regexp.Compile(`^Test`)
	}

	return nil, fmt.Errorf("comparing against a commit hash is not yet implemented")
}

func (p *Package) hasDirtyWorktree() (dirty bool, files []string, err error) {
	if r, err := p.Repository(); err == nil {
		p.repo = r
	} else {
		return dirty, files, err
	}

	if w, err := p.repo.Worktree(); err == nil {
		p.worktree = w
	} else {
		return dirty, files, err
	}

	if s, err := p.worktree.Status(); err == nil {
		p.status = s
	} else {
		return dirty, files, err
	}

	for file, status := range p.status {
		if status.Worktree != git.Unmodified {
			files = append(files, filepath.Join(p.repoDir, file))
		}
	}

	files = filter(files, func(s string) bool {
		if filepath.Dir(strings.TrimPrefix(s, p.Dir)) == "/" {
			return path.Ext(s) == ".go"
		}
		return false
	})

	return len(files) > 0, files, nil
}

func filter(vs []string, f func(string) bool) []string {
	r := []string{}
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
