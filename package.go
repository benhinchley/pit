package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

type Package struct {
	Name string
}

func packages() ([]Package, error) {
	result := []Package{}

	cmd := exec.Command("go", "list", "./...")
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return result, fmt.Errorf("unable to run \"go list ./...\": %s", err)
	}

	pkgs := strings.Split(strings.TrimSpace(out.String()), "\n")
	for _, pkg := range pkgs {
		result = append(result, Package{
			Name: pkg,
		})
	}

	return result, nil
}

func (p Package) Sources() ([]string, error) {
	result := []string{}
	cmd := exec.Command("go", "list", "-f", "'{{ join .GoFiles \",\" }}'", p.Name)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return result, fmt.Errorf("unable to run \"go list -f '{{ join .GoFiles \",\" }}' %s\": %s", p.Name, err)
	}

	re, err := regexp.Compile(`(\w+.go)`)
	if err != nil {
		return result, fmt.Errorf("unable to compile regex `(\\w+.go)`: %s", err)
	}
	for _, match := range re.FindAllString(strings.TrimSpace(out.String()), -1) {
		result = append(result, match)
	}

	return result, nil
}

func (p Package) Tests() ([]string, error) {
	result := []string{}

	cmd := exec.Command("go", "list", "-f", "'{{ join .TestGoFiles \",\" }}'", p.Name)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return result, fmt.Errorf("unable to run \"go list -f '{{ join .TestGoFiles \",\" }}' %s\": %s", p.Name, err)
	}

	re, err := regexp.Compile(`(\w+.go)`)
	if err != nil {
		return result, fmt.Errorf("unable to compile regex `(\\w+.go)`: %s", err)
	}
	for _, match := range re.FindAllString(strings.TrimSpace(out.String()), -1) {
		result = append(result, match)
	}

	return result, nil
}

func namedDiffFiles(files []string) ([]string, error) {
	args := []string{"add", "-N"}
	args = append(args, files...)

	cmd := exec.Command("git", args...)

	if err := cmd.Run(); err != nil {
		return []string{}, fmt.Errorf("unable to run \"git add -N %s\": %s", strings.Join(args, " "), err)
	}

	args = []string{"diff", "--name-only", "HEAD"}
	cmd = exec.Command("git", args...)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return []string{}, fmt.Errorf("unable to run \"git diff --name-only HEAD\": %s", err)
	}

	r := strings.Split(strings.TrimSpace(out.String()), "\n")

	cmd = exec.Command("git", "reset")

	if err := cmd.Run(); err != nil {
		return []string{}, fmt.Errorf("unable to run \"git reset\": %s", err)
	}

	return r, nil
}
