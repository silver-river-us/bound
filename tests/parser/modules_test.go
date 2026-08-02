package parser_test

import (
	. "bound/src/parser"
	"strings"
	"testing"

	"bound/src/model"
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

func TestArchitectureEntrypointDerivesRootImplementationFile(t *testing.T) {
	architecture, err := Parse(strings.NewReader(`
architecture Example do
  implementation go "./"
  entrypoint :main
  context Reporting do
  end
end
`))
	if err != nil {
		t.Fatalf("parse architecture: %v", err)
	}
	if err := architecture.Validate(); err != nil {
		t.Fatalf("validate architecture: %v", err)
	}
	if len(architecture.Files) != 1 {
		t.Fatalf("files = %#v, want one root entrypoint", architecture.Files)
	}
	entrypoint := architecture.Files[0]
	if !entrypoint.RootEntrypoint || !entrypoint.Explicit || entrypoint.Node != "" || entrypoint.Path != "main.go" {
		t.Fatalf("entrypoint mapping = %#v, want root main.go", entrypoint)
	}
}

func TestModulesDeclareSourceFilesAndEntrypointPaths(t *testing.T) {
	architecture, err := Parse(strings.NewReader(`
architecture Example do
  implementation go "./"
  context Reporting do
    module Reporting do
      module Command do
        module DailyReport do
          files [:report, :summary]
          entrypoint DailyReport
        end
      end
    end
  end
end
`))
	if err != nil {
		t.Fatalf("parse architecture: %v", err)
	}
	module := architecture.Modules["Reporting.Command.DailyReport"]
	if module == nil || len(module.Files) != 2 || module.Files[0] != "report" || module.Files[1] != "summary" {
		t.Fatalf("module files = %#v", module)
	}
	if err := architecture.Validate(); err != nil {
		t.Fatalf("validate architecture: %v", err)
	}
	if len(architecture.Files) != 3 {
		t.Fatalf("files = %d, want 3", len(architecture.Files))
	}
	entrypoint := architecture.Files[2]
	if !entrypoint.EntryPoint || entrypoint.EntryPointName != "DailyReport" || entrypoint.Node != "Reporting.Command.DailyReport" || entrypoint.Path != "reporting/command/daily_report/main.go" {
		t.Fatalf("entrypoint mapping = %#v", entrypoint)
	}
}

func TestFilesRejectNonAtomEntries(t *testing.T) {
	_, err := Parse(strings.NewReader(`
architecture Example do
  implementation go "./"
  context Reporting do
    module Reporting do
      files ["report.go"]
    end
  end
end
`))
	if err == nil || !strings.Contains(err.Error(), "invalid file atom") {
		t.Fatalf("error = %v, want invalid file atom", err)
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
	if err == nil || !strings.Contains(err.Error(), "expected import, interface, module, exposes, or end") {
		t.Fatalf("error = %v, want qualified module declaration rejection", err)
	}
}

func TestSiblingModulesRejectConventionalFolderCollision(t *testing.T) {
	_, err := Parse(strings.NewReader(`
architecture Example do
  implementation go "./"
  context Reporting do
    module FooBar do
    end
    module Foo_Bar do
    end
  end
end
`))
	if err == nil || !strings.Contains(err.Error(), "folder name collides") {
		t.Fatalf("error = %v, want folder collision", err)
	}
}
