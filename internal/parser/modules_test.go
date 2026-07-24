package parser

import (
	"strings"
	"testing"
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
	if err := architecture.Validate(); err != nil {
		t.Fatalf("validate architecture: %v", err)
	}
}
