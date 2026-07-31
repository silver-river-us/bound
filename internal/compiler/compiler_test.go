package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bound/internal/model"
)

func TestCompileResolvesImplementationRootRelativeToArchitecture(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example\n\ngo 1.22\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "lib", "service.go"), []byte("package lib\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	architecturePath := filepath.Join(root, "architecture.bo")
	const source = `architecture Example do
  implementation go "."
  context Lib do
    module Lib do
      files [:service]
    end
  end
end
`
	if err := os.WriteFile(architecturePath, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}

	program, err := Compile(architecturePath, Options{})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if program.SourceRoot != root {
		t.Fatalf("source root = %q, want %q", program.SourceRoot, root)
	}
}

func TestCompileReportsModelValidationErrors(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "architecture.bo")
	const source = `architecture Example do
  implementation ruby "."
end
`
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Compile(path, Options{})
	if err == nil || !strings.Contains(err.Error(), "unsupported implementation language ruby") {
		t.Fatalf("error = %v, want unsupported language diagnostic", err)
	}
	var compilationError *Error
	if !asCompilationError(err, &compilationError) || len(compilationError.Diagnostics) != 1 || compilationError.Diagnostics[0].Phase != "validate" {
		t.Fatalf("error diagnostics = %#v", compilationError)
	}
}

func TestCompileCanStopAfterModelValidation(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "architecture.bo")
	const source = `architecture Example do
  implementation go "."
end
`
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Compile(path, Options{SkipImplementation: true}); err != nil {
		t.Fatalf("compile without backend check: %v", err)
	}
}

func TestProgramJSONContainsResolvedIR(t *testing.T) {
	program := &Program{
		Path:       "/tmp/architecture.bo",
		SourceRoot: "/tmp",
		Architecture: &model.Architecture{
			Name:           "Example",
			Implementation: model.Implementation{Language: "go", Locator: "."},
			Contexts:       map[string]*model.Context{},
			Objects:        map[string]*model.Object{},
			Modules:        map[string]*model.Module{},
		},
	}
	encoded, err := program.JSON()
	if err != nil {
		t.Fatalf("encode JSON: %v", err)
	}
	output := string(encoded)
	for _, expected := range []string{`"schema_version": 1`, `"source_root": "/tmp"`, `"name": "Example"`, `"language": "go"`} {
		if !strings.Contains(output, expected) {
			t.Errorf("JSON does not contain %q: %s", expected, output)
		}
	}
}

func asCompilationError(err error, target **Error) bool {
	value, ok := err.(*Error)
	if ok {
		*target = value
	}
	return ok
}
