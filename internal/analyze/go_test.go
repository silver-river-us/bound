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
	file, err := os.Open(filepath.Join(root, "..", "..", "examples", "commerce.bo"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	a, err := parser.Parse(file)
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
