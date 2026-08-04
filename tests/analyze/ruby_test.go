package analyze_test

import (
	. "bound/src/infrastructure/analyze"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bound/src/lib/model"
)

func TestRubyValidatesMappedSourceOwnership(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "api", "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "api", "app", "application.rb"), []byte("class Application; end\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	architecture := &model.Architecture{
		Implementation: model.Implementation{Language: "ruby"},
		Contexts:       map[string]*model.Context{},
		Modules: map[string]*model.Module{
			"Platform.API": {Qualified: "Platform.API", Context: "Platform"},
		},
		Files: []model.FileMapping{{Path: "api/app/application.rb", Node: "Platform.API"}},
	}
	if err := Ruby(root, architecture); err != nil {
		t.Fatalf("Ruby analysis: %v", err)
	}
}

func TestRubyRejectsUnmappedRubySource(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "application.rb"), []byte("class Application; end\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	architecture := &model.Architecture{
		Implementation: model.Implementation{Language: "ruby"},
		Contexts:       map[string]*model.Context{},
		Modules:        map[string]*model.Module{},
	}
	if err := Ruby(root, architecture); err == nil || !strings.Contains(err.Error(), "has no architecture mapping") {
		t.Fatalf("error = %v, want unmapped Ruby source error", err)
	}
}
