package analyze

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/silver-river-us/bound/internal/model"
	"github.com/silver-river-us/bound/internal/parser"
)

func exampleArchitecture(t *testing.T) *model.Architecture {
	t.Helper()
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	a, err := parser.ParseFile(filepath.Join(root, "..", "..", "examples", "commerce.bo"))
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
	return a
}

func TestGoAcceptsDeclaredExampleDependencies(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := Go(filepath.Join(root, "..", "..", "examples", "commerce"), exampleArchitecture(t)); err != nil {
		t.Fatal(err)
	}
}

func TestGoRejectsUndeclaredExampleDependency(t *testing.T) {
	a := exampleArchitecture(t)
	a.Relations = []model.Relation{{From: "App", To: "Web"}}
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := Go(filepath.Join(root, "..", "..", "examples", "commerce"), a); err == nil {
		t.Fatal("expected undeclared dependency error")
	}
}

func TestGoRejectsUnmappedSourceFile(t *testing.T) {
	a := exampleArchitecture(t)
	a.Files = append(a.Files[:0], a.Files[1:]...)
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := Go(filepath.Join(root, "..", "..", "examples", "commerce"), a); err == nil {
		t.Fatal("expected unmapped source file error")
	}
}

func TestGoRequiresExplicitEntrypoint(t *testing.T) {
	a := exampleArchitecture(t)
	for index := range a.Files {
		a.Files[index].EntryPoint = false
	}
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := Go(filepath.Join(root, "..", "..", "examples", "commerce"), a); err == nil {
		t.Fatal("expected explicit entrypoint error")
	}
}

func TestGoRejectsMultipleDeclarationsPerFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "multiple.go")
	if err := os.WriteFile(path, []byte("package example\n\ntype A struct{}\ntype B struct{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateOneDeclaration(path); err == nil {
		t.Fatal("expected multiple declaration error")
	}
}
