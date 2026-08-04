package compiler_test

import (
	. "bound/src/lib/compiler"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bound/src/lib/model"
)

type recordingBackend struct {
	called bool
}

func (backend *recordingBackend) Language() string { return "typescript" }
func (backend *recordingBackend) Analyze(root string, architecture *model.Architecture) error {
	backend.called = true
	return nil
}

func TestCompileUsesInjectedBackendRegistry(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "architecture.bo")
	const source = `architecture Example do
	  implementation typescript "."
	end
	`
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	backend := &recordingBackend{}
	registry := NewRegistry(backend)
	if _, err := Compile(path, Options{Backends: &registry}); err != nil {
		t.Fatalf("compile with injected backend: %v", err)
	}
	if !backend.called {
		t.Fatal("injected backend was not called")
	}
}

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

func TestCompileChecksPythonImplementationTree(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "orders"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "orders", "service.py"), []byte("class Service: pass\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	architecturePath := filepath.Join(root, "architecture.bo")
	const source = `architecture Example do
  implementation python "."
  context Orders do
    module Orders do
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
		t.Fatalf("compile Python architecture: %v", err)
	}
	if program.Architecture.Implementation.Language != "python" {
		t.Fatalf("language = %q, want python", program.Architecture.Implementation.Language)
	}
}

func TestCompileReportsModelValidationErrors(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "architecture.bo")
	const source = `architecture Example do
  implementation future "."
end
`
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Compile(path, Options{})
	if err == nil || !strings.Contains(err.Error(), "unsupported implementation language future") {
		t.Fatalf("error = %v, want unsupported language diagnostic", err)
	}
	var compilationError *Error
	if !asCompilationError(err, &compilationError) || len(compilationError.Diagnostics) != 1 || compilationError.Diagnostics[0].Phase != "validate" {
		t.Fatalf("error diagnostics = %#v", compilationError)
	}
}

func TestCompileModelDiagnosticsIncludeLocationAndSuggestion(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "architecture.bo")
	const source = `architecture Example do
  implementation go "."
  context Orders do
    exposes Order
    interface Orders do
    end
  end
end
`
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Compile(path, Options{SkipImplementation: true})
	var compilationError *Error
	if !asCompilationError(err, &compilationError) || len(compilationError.Diagnostics) != 1 {
		t.Fatalf("error = %#v", err)
	}
	diagnostic := compilationError.Diagnostics[0]
	if diagnostic.Line != 4 || diagnostic.Column != 1 {
		t.Fatalf("location = %d:%d, want 4:1", diagnostic.Line, diagnostic.Column)
	}
	if diagnostic.Suggestion != `did you mean interface "Orders"?` {
		t.Fatalf("suggestion = %q", diagnostic.Suggestion)
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
