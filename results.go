package pit

import (
	"fmt"
	"time"

	"github.com/benhinchley/pit/internals/testparser"
)

type PackageTestResult struct {
	Name     string         `json:"name"`
	Status   string         `json:"status"`
	Duration time.Duration  `json:"duration"`
	Coverage float32        `json:"coverage"`
	Summary  string         `json:"summary"`
	Tests    []*PackageTest `json:"tests"`
	Errors   []*Failure     `json:"errors"`
}

type PackageTest struct {
	Name     string        `json:"name"`
	Status   string        `json:"status"`
	Duration time.Duration `json:"duration"`
	Output   []string      `json:"ouput"`
}

type Failure struct {
	File    string `json:"filename"`
	Row     int    `json:"row"`
	Column  int    `json:"column"`
	Message string `json:"message"`
}

func (f *Failure) String() string {
	return fmt.Sprintf("%s:%d:%d: %s", f.File, f.Row, f.Column, f.Message)
}

// fromTestparser converts *testparser.PackageResult into *PackageTestResult
// This is so our internal testparser package types don't get exposed as part of the public API
func fromTestparser(r *testparser.PackageResult) *PackageTestResult {
	pr := &PackageTestResult{}
	pr.Name = r.Name
	pr.Status = r.Status.String()
	pr.Summary = r.Summary
	pr.Duration = r.Duration
	pr.Coverage = r.Coverage

	for _, test := range r.Tests {
		pr.Tests = append(pr.Tests, &PackageTest{
			Name:     test.Name,
			Duration: test.Duration,
			Status:   test.Status.String(),
			Output:   test.Output,
		})
	}

	for _, failure := range r.Errors {
		pr.Errors = append(pr.Errors, &Failure{
			File:    failure.File,
			Row:     failure.Row,
			Column:  failure.Column,
			Message: failure.Message,
		})
	}

	return pr
}
