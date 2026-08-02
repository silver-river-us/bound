package bound_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"bound"
)

func TestPublicCompileAPI(t *testing.T) {
	path := filepath.Join(t.TempDir(), "architecture.bo")
	const source = `architecture Example do
  implementation go "."
end
`
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}

	program, err := bound.Compile(path, bound.Options{SkipImplementation: true})
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if program.Path() != path {
		t.Fatalf("program path = %q, want %q", program.Path(), path)
	}
	ir, err := program.JSON()
	if err != nil {
		t.Fatalf("encode IR: %v", err)
	}
	if len(ir) == 0 {
		t.Fatal("compiler IR is empty")
	}
	if bound.SchemaVersion != 1 {
		t.Fatalf("schema version = %d, want 1", bound.SchemaVersion)
	}
}

func TestPublicCompileAPIProvidesTypedDiagnostics(t *testing.T) {
	path := filepath.Join(t.TempDir(), "architecture.bo")
	if err := os.WriteFile(path, []byte("architecture Example do\n  implementation future \".\"\nend\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := bound.Compile(path, bound.Options{})
	var compilationError *bound.Error
	if !errors.As(err, &compilationError) {
		t.Fatalf("error type = %T, want *bound.Error", err)
	}
	if diagnostics := compilationError.Diagnostics(); len(diagnostics) != 1 || diagnostics[0].Phase != "validate" || diagnostics[0].Line != 2 || diagnostics[0].Column != 1 {
		t.Fatalf("diagnostics = %#v, want one validate diagnostic at 2:1", diagnostics)
	}
}
