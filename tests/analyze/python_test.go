package analyze_test

import (
	. "bound/src/analyze"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bound/src/model"
)

func TestPythonValidatesMappedSourceOwnershipAndModulePaths(t *testing.T) {
	root := t.TempDir()
	moduleRoot := filepath.Join(root, "orders")
	if err := os.MkdirAll(moduleRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(moduleRoot, "service.py"), []byte("class Service: pass\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	architecture := &model.Architecture{
		Implementation: model.Implementation{Language: "python"},
		Contexts:       map[string]*model.Context{},
		Modules: map[string]*model.Module{
			"Orders": {Name: "Orders", Qualified: "Orders", Context: "Orders"},
		},
		Files: []model.FileMapping{{Path: "orders/service.py", Node: "Orders"}},
	}
	if err := Python(root, architecture); err != nil {
		t.Fatalf("Python analysis: %v", err)
	}
}

func TestPythonRejectsUnmappedSourceAndWrongModulePath(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "orders"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"orders/service.py", "orders/other.py"} {
		if err := os.WriteFile(filepath.Join(root, path), []byte("value = 1\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	architecture := &model.Architecture{
		Implementation: model.Implementation{Language: "python"},
		Contexts:       map[string]*model.Context{},
		Modules: map[string]*model.Module{
			"Orders": {Name: "Orders", Qualified: "Orders", Context: "Orders"},
		},
		Files: []model.FileMapping{{Path: "orders/service.py", Node: "Orders"}},
	}
	if err := Python(root, architecture); err == nil || !strings.Contains(err.Error(), "has no architecture mapping") {
		t.Fatalf("error = %v, want unmapped Python source error", err)
	}

	if err := os.WriteFile(filepath.Join(root, "service.py"), []byte("value = 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	architecture.Files = []model.FileMapping{{Path: "service.py", Node: "Orders"}}
	if err := Python(root, architecture); err == nil || !strings.Contains(err.Error(), "outside module Orders folder") {
		t.Fatalf("error = %v, want conventional module path error", err)
	}
}
