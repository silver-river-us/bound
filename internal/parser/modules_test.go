package parser

import (
	"strings"
	"testing"

	"github.com/silver-river-us/bound/internal/model"
)

func TestNestedModulesDeclareContractsDependenciesAndEntrypoints(t *testing.T) {
	architecture, err := Parse(strings.NewReader(`
architecture Example do
  implementation go "./"

  context Reporting do
    interface Reports do
      behavior render() returns string
    end
    exposes Reports

    module Reporting do
      module Renderer do
        implements Reports
        uses Helpers
        entrypoint DailyReport

        module Helpers do
        end
      end
    end
  end
end
`))
	if err != nil {
		t.Fatalf("parse architecture: %v", err)
	}

	renderer := architecture.Modules["Reporting.Renderer"]
	if renderer == nil {
		t.Fatal("renderer module was not registered")
	}
	if renderer.Implements != "Reports" {
		t.Fatalf("implements = %q, want Reports", renderer.Implements)
	}
	if !renderer.Uses["Helpers"] {
		t.Fatal("renderer does not use Helpers")
	}
	if !renderer.Entrypoints["DailyReport"] {
		t.Fatal("renderer does not declare DailyReport entrypoint")
	}
	if child := renderer.Modules["Helpers"]; child == nil || child.Parent != renderer.Qualified {
		t.Fatal("nested helper module has the wrong parent")
	}
	architecture.Files = []model.FileMapping{{
		Path:           "reporting/renderer/cmd/daily-report/main.go",
		Node:           renderer.Qualified,
		EntryPoint:     true,
		EntryPointName: "DailyReport",
	}}
	if err := architecture.Validate(); err != nil {
		t.Fatalf("validate architecture: %v", err)
	}
}

func TestModuleDeclarationRejectsQualifiedName(t *testing.T) {
	_, err := Parse(strings.NewReader(`
architecture Example do
  implementation go "./"
  context Reporting do
    module Reporting.Activity do
    end
  end
end
`))
	if err == nil || !strings.Contains(err.Error(), "expected interface, module, exposes, or end") {
		t.Fatalf("error = %v, want qualified module declaration rejection", err)
	}
}
