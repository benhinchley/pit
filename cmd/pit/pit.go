package main

import (
	"bytes"
	"flag"
	"fmt"
	"strings"

	"github.com/intel/tfortools"

	"github.com/benhinchley/cmd"
	"github.com/benhinchley/pit"
	"github.com/benhinchley/pit/internals/testparser"
)

var defaultTemplate = `
{{tablex (cols . "Name" "Status" "Duration" "Coverage" "Summary") 8 8 4 "Package" }}

{{- range . -}}{{if gt (len .Tests) 0}}
Package: {{println .Name}}
{{- cols .Tests "Name" "Duration" "Status" | table}}
{{- end}}{{- end}}
`

type pitCommand struct {
	template string
	json     bool
	all      bool
}

func (cmd *pitCommand) Name() string { return "pit" }
func (cmd *pitCommand) Args() string { return "[-f format] [-json]" }
func (cmd *pitCommand) Desc() string { return "smartish wrapper around go test" }
func (cmd *pitCommand) Help() string { return "TODO" }
func (cmd *pitCommand) Register(fs *flag.FlagSet) {
	fs.StringVar(&cmd.template, "f", strings.TrimSpace(defaultTemplate), "output template")
	fs.BoolVar(&cmd.all, "all", false, "run all tests.")
	fs.BoolVar(&cmd.json, "json", false, "print test result data in json format.")
}

func (cmd *pitCommand) Run(ctx cmd.Context, args []string) error {
	wd := ctx.(*context).WorkingDir

	pkgs, err := pit.FindPackages(wd)
	if err != nil {
		return fmt.Errorf("unable to find packages: %v", err)
	}

	var results []*testparser.PackageResult
	for _, pkg := range pkgs {
		r, err := pkg.RunTests(cmd.all)
		if err != nil {
			return fmt.Errorf("unable to run test for \"%s\": %v", pkg.ImportPath, err)
		}
		results = append(results, r)
	}

	tmpl := cmd.template
	if cmd.json {
		tmpl = "{{tojson .}}"
	}

	t, err := tfortools.CreateTemplate("pit", tmpl, ctx.(*context).TemplateConfig)
	if err != nil {
		return err
	}

	var out bytes.Buffer
	t.Execute(&out, results)
	ctx.Stdout().Print(out.String())

	return nil
}
