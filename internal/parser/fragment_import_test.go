package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestContextImportsInterfaceFragment(t *testing.T) {
	directory := t.TempDir()
	architecturePath := filepath.Join(directory, "example.bo")
	fragmentPath := filepath.Join(directory, "reports.bo")

	writeTestFile(t, fragmentPath, `
interface Reports do
  value Snapshot do
    state :created_at :timestamp
  end
  behavior render(snapshot Snapshot) returns string
end
`)
	writeTestFile(t, architecturePath, `
architecture Example do
  implementation go "./"
  context Reporting do
    import contracts from "reports.bo"
    exposes Reports
  end
end
`)

	architecture, err := ParseFile(architecturePath)
	if err != nil {
		t.Fatalf("parse architecture: %v", err)
	}
	contract := architecture.Contexts["Reporting"].Interfaces["Reports"]
	if contract == nil || contract.Types["Snapshot"] == nil {
		t.Fatal("imported interface contract was not merged into its context")
	}
	if err := architecture.Validate(); err != nil {
		t.Fatalf("validate architecture: %v", err)
	}
}

func TestContextFragmentImportsAnotherFragment(t *testing.T) {
	directory := t.TempDir()
	architecturePath := filepath.Join(directory, "example.bo")
	writeTestFile(t, filepath.Join(directory, "types.bo"), `
interface Types do
  value Identifier do
    state :value :string
  end
end
`)
	writeTestFile(t, filepath.Join(directory, "reports.bo"), `
import contracts from "types.bo"
interface Reports do
  behavior find(id Types.Identifier) returns string
end
`)
	writeTestFile(t, architecturePath, `
architecture Example do
  implementation go "./"
  context Reporting do
    import contracts from "reports.bo"
    exposes Reports
  end
end
`)

	architecture, err := ParseFile(architecturePath)
	if err != nil {
		t.Fatalf("parse architecture: %v", err)
	}
	if architecture.Contexts["Reporting"].Interfaces["Types"] == nil {
		t.Fatal("recursively imported interface was not merged")
	}
}

func TestContextFragmentRejectsModules(t *testing.T) {
	directory := t.TempDir()
	architecturePath := filepath.Join(directory, "example.bo")
	writeTestFile(t, filepath.Join(directory, "invalid.bo"), `
module Internal do
end
`)
	writeTestFile(t, architecturePath, `
architecture Example do
  implementation go "./"
  context Reporting do
    import contracts from "invalid.bo"
  end
end
`)

	_, err := ParseFile(architecturePath)
	if err == nil || !strings.Contains(err.Error(), "may only contain imports and interfaces") {
		t.Fatalf("error = %v, want fragment module rejection", err)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
