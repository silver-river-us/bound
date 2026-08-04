package analyze_test

import (
	. "bound/src/infrastructure/analyze"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bound/src/lib/model"
)

func TestGoAnalyzesNestedWorkspaceModules(t *testing.T) {
	root := t.TempDir()
	writeGoTestFile(t, filepath.Join(root, "go.work"), "go 1.22\n\nuse (\n\t./core\n\t./feature\n)\n")
	writeGoTestFile(t, filepath.Join(root, "core", "go.mod"), "module example/core\n\ngo 1.22\n")
	writeGoTestFile(t, filepath.Join(root, "core", "core.go"), "package core\n\nfunc Value() int { return 1 }\n")
	writeGoTestFile(t, filepath.Join(root, "feature", "go.mod"), "module example/feature\n\ngo 1.22\n")
	writeGoTestFile(t, filepath.Join(root, "feature", "feature.go"), "package feature\n\nimport \"example/core\"\n\nfunc Value() int { return core.Value() }\n")

	architecture := &model.Architecture{
		Implementation: model.Implementation{Language: "go"},
		Contexts:       map[string]*model.Context{},
		Modules: map[string]*model.Module{
			"Core":    {Name: "Core", Qualified: "Core", Context: "Domain"},
			"Feature": {Name: "Feature", Qualified: "Feature", Context: "Boundary"},
		},
		Files: []model.FileMapping{
			{Path: "core/core.go", Node: "Core", Explicit: true},
			{Path: "feature/feature.go", Node: "Feature", Explicit: true},
		},
	}

	err := Go(root, architecture)
	if err == nil || !strings.Contains(err.Error(), "example/feature (Feature) imports example/core (Core)") {
		t.Fatalf("error = %v, want cross-module dependency error", err)
	}
}

func TestGoAnalyzesImplementationRootOutsideRepository(t *testing.T) {
	root := t.TempDir()
	writeGoTestFile(t, filepath.Join(root, "go.mod"), "module example/outside\n\ngo 1.22\n")
	writeGoTestFile(t, filepath.Join(root, "service", "service.go"), "package service\n\nfunc Value() int { return 1 }\n")

	architecture := &model.Architecture{
		Implementation: model.Implementation{Language: "go"},
		Contexts:       map[string]*model.Context{},
		Modules: map[string]*model.Module{
			"Service": {Name: "Service", Qualified: "Service", Context: "ServiceContext"},
		},
		Files: []model.FileMapping{{Path: "service/service.go", Node: "Service"}},
	}
	if err := Go(root, architecture); err != nil {
		t.Fatalf("Go analysis: %v", err)
	}
}

func writeGoTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
