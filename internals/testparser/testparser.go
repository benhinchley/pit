// Package testparser provides a parser for go test output
package testparser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Regular Expressions for various parts of the go test output
var (
	RegexTest     = regexp.MustCompile(`=== RUN (.+)`)
	RegexStatus   = regexp.MustCompile(`^\s*--- (PASS|FAIL|SKIP): (.+) \((\d+\.\d+)(?: seconds|s)\)$`)
	RegexCoverage = regexp.MustCompile(`^coverage:\s+(\d+\.\d+)%\s+of\s+statements(?:\sin\s.+)?$`)
	RegexResult   = regexp.MustCompile(`^(ok|FAIL)\s+([^ ]+)\s+(?:(\d+\.\d+)s|(\[\w+ failed]))(?:\s+coverage:\s+(\d+\.\d+)%\sof\sstatements(?:\sin\s.+)?)?$`)
	RegexOuput    = regexp.MustCompile(`(    )*\t(.*)`)
	RegexSummary  = regexp.MustCompile(`^(PASS|FAIL|SKIP)$`)
	RegexFail     = regexp.MustCompile(`(.*\w+.go)*:(\d+):(\d+):(.*\w+)`)
)

type PackageResult struct {
	Name     string        `json:"name"`
	Status   Status        `json:"status"`
	Duration time.Duration `json:"duration"`
	Coverage float32       `json:"coverage"`
	Summary  string        `json:"summary"`
	Tests    []*Test       `json:"tests"`
	Errors   []*FailLine   `json:"errors"`
}

type FailLine struct {
	File    string `json:"filename"`
	Row     int    `json:"row"`
	Column  int    `json:"column"`
	Message string `json:"message"`
}

func (fl *FailLine) String() string {
	return fmt.Sprintf("%s:%d:%d: %s", fl.File, fl.Row, fl.Column, fl.Message)
}

type Test struct {
	Name     string        `json:"name"`
	Status   Status        `json:"status"`
	Duration time.Duration `json:"duration"`
	Output   []string      `json:"output"`
}

type Status int

func (s Status) String() string {
	switch s {
	case StatusFail:
		return "FAIL"
	case StatusPass:
		return "PASS"
	default:
		return "SKIP"
	}
}

func (s *Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

const (
	StatusFail Status = iota
	StatusPass
	StatusSkip
)

// var debug = log.New(os.Stderr, "testparser: ", 0)

// Parse ...
func Parse(r io.Reader) ([]*PackageResult, error) {
	var pkgs []*PackageResult
	tests := []*Test{}
	scanner := bufio.NewScanner(r)

	duration := func(d string) string {
		switch d {
		case "":
			return "0s"
		default:
			return d + "s"
		}
	}

	var (
		currentTestName    string
		coveragePercentage float32
		fails              = []*FailLine{}
	)

	for scanner.Scan() {
		line := scanner.Text()

		if RegexTest.MatchString(line) {
			matches := RegexTest.FindStringSubmatch(line)
			// debug.Println("RegexTest Matches:", litter.Sdump(matches))
			currentTestName = strings.TrimSpace(matches[1])
			tests = append(tests, &Test{Name: currentTestName})
		} else if RegexResult.MatchString(line) {
			matches := RegexResult.FindStringSubmatch(line)
			// debug.Println("RegexResult Matches:", litter.Sdump(matches))

			d, err := time.ParseDuration(duration(matches[3]))
			if err != nil {
				return nil, fmt.Errorf("testparser: RegexResult: unable to parse duration: %v", err)
			}

			pkg := &PackageResult{
				Name:     matches[2],
				Status:   toStatus(matches[1]),
				Coverage: coveragePercentage,
				Duration: d,
				Tests:    tests,
			}

			if matches[4] != "" {
				pkg.Summary = matches[4]
			}

			if len(fails) > 0 {
				pkg.Errors = fails
			}

			pkgs = append(pkgs, pkg)
			tests = []*Test{}
			currentTestName = ""
			coveragePercentage = 0.0
		} else if RegexStatus.MatchString(line) {
			matches := RegexStatus.FindStringSubmatch(line)
			// debug.Println("RegexStatus Matches:", litter.Sdump(matches))

			if t := getTest(tests, currentTestName); t != nil {
				d, err := time.ParseDuration(duration(matches[3]))
				if err != nil {
					return nil, fmt.Errorf("testparser: RegexStatus: unable to parse duration: %v", err)
				}

				t.Status = toStatus(matches[1])
				t.Duration = d
			}
		} else if RegexCoverage.MatchString(line) {
			matches := RegexCoverage.FindStringSubmatch(line)
			// debug.Println("RegexCoverage Matches:", litter.Sdump(matches))

			p, _ := strconv.ParseFloat(matches[1], 32)
			coveragePercentage = float32(p)
		} else if RegexOuput.MatchString(line) {
			matches := RegexOuput.FindStringSubmatch(line)
			// debug.Println("RegexOutput Matches:", litter.Sdump(matches))

			if t := getTest(tests, currentTestName); t != nil {
				t.Output = append(t.Output, matches[2])
			}
		} else if RegexFail.MatchString(line) {
			matches := RegexFail.FindStringSubmatch(line)
			// debug.Println("RegexFail Matches:", litter.Sdump(matches))
			row, _ := strconv.Atoi(matches[2])
			column, _ := strconv.Atoi(matches[3])

			fails = append(fails, &FailLine{
				File:    matches[1],
				Row:     row,
				Column:  column,
				Message: strings.TrimSpace(matches[4]),
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("testparser: unable to parse input: %v", err)
	}
	return pkgs, nil
}

func getTest(tests []*Test, name string) *Test {
	for _, test := range tests {
		if test.Name == name {
			return test
		}
	}
	return nil
}

func toStatus(s string) Status {
	switch strings.ToLower(s) {
	case "fail", "failed":
		return StatusFail
	case "pass", "ok":
		return StatusPass
	default:
		return StatusSkip
	}
}
