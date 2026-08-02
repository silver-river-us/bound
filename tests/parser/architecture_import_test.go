package parser_test

import (
	. "bound/src/parser"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchitectureImportRequiresNamedArchitecture(t *testing.T) {
	directory := t.TempDir()
	writeTestFile(t, filepath.Join(directory, "shared.bo"), `
architecture Shared do
  implementation go "./"
  context SharedContext do
  end
end
`)
	writeTestFile(t, filepath.Join(directory, "root.bo"), `
architecture Root do
  implementation go "./"
  import Shared from "shared.bo"
end
`)

	architecture, err := ParseFile(filepath.Join(directory, "root.bo"))
	if err != nil {
		t.Fatalf("parse architecture: %v", err)
	}
	if architecture.Contexts["SharedContext"] == nil {
		t.Fatal("named architecture import was not merged")
	}
}

func TestArchitectureImportRejectsWrongName(t *testing.T) {
	directory := t.TempDir()
	writeTestFile(t, filepath.Join(directory, "shared.bo"), `
architecture Shared do
  implementation go "./"
end
`)
	writeTestFile(t, filepath.Join(directory, "root.bo"), `
architecture Root do
  implementation go "./"
  import Wrong from "shared.bo"
end
`)

	_, err := ParseFile(filepath.Join(directory, "root.bo"))
	if err == nil || !strings.Contains(err.Error(), "imported as Wrong") {
		t.Fatalf("error = %v, want wrong architecture import name", err)
	}
}

func TestMapImportRequiresNamedArchitecture(t *testing.T) {
	directory := t.TempDir()
	writeTestFile(t, filepath.Join(directory, "root.bo"), `
architecture Root do
  implementation go "./"
  import Root from "root.bom"
end
`)
	writeTestFile(t, filepath.Join(directory, "root.bom"), `
map Root do
  "root.go" -> Root.App
end
`)

	architecture, err := ParseFile(filepath.Join(directory, "root.bo"))
	if err != nil {
		t.Fatalf("parse architecture: %v", err)
	}
	if len(architecture.Files) != 1 || architecture.Files[0].Path != "root.go" {
		t.Fatalf("files = %#v, want imported map file", architecture.Files)
	}
}
