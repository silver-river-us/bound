package parser

import (
	"strings"
	"testing"
)

func TestInterfaceOwnsItsTypes(t *testing.T) {
	architecture, err := Parse(strings.NewReader(`
architecture Example do
  implementation go "./"

  context Reporting do
    interface Reports do
      entity Organization do
        state :login :string
      end

      value Snapshot do
        state :organizations :Reports.Organization[]
      end

      behavior render(snapshot Snapshot) returns string
    end
    exposes Reports
  end
end
`))
	if err != nil {
		t.Fatalf("parse architecture: %v", err)
	}

	contract := architecture.Contexts["Reporting"].Interfaces["Reports"]
	if len(contract.Types) != 2 {
		t.Fatalf("interface types = %d, want 2", len(contract.Types))
	}
	if contract.Types["Organization"].Kind != "entity" {
		t.Fatalf("Organization kind = %q, want entity", contract.Types["Organization"].Kind)
	}
	if got := contract.Types["Snapshot"].Attributes["organizations"].Type; got != "Reports.Organization[]" {
		t.Fatalf("Snapshot.organizations type = %q, want Reports.Organization[]", got)
	}
	if err := architecture.Validate(); err != nil {
		t.Fatalf("validate architecture: %v", err)
	}
}
