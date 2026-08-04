package analyze

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bound/src/lib/model"
)

// Python validates explicit source ownership for Python repositories. Like the
// Ruby backend, it does not infer dependencies from imports; it checks the
// source map and the conventional module layout declared by the architecture.
func Python(root string, architecture *model.Architecture) error {
	if architecture.Implementation.Language != "python" {
		return fmt.Errorf("architecture implementation must use Python")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve Python root: %w", err)
	}

	mappings := make(map[string]model.FileMapping, len(architecture.Files))
	for _, module := range architecture.Modules {
		location := moduleLocation(root, architecture, module)
		info, statErr := os.Stat(location)
		if statErr != nil {
			if os.IsNotExist(statErr) && moduleHasOnlyExplicitEntrypoints(architecture, module) {
				continue
			}
			return fmt.Errorf("module %s folder %s: %w", module.Qualified, filepath.ToSlash(location), statErr)
		}
		if !info.IsDir() {
			return fmt.Errorf("module %s path %s must be a folder", module.Qualified, filepath.ToSlash(location))
		}
	}

	for _, mapping := range architecture.Files {
		if mapping.Node == "" && !mapping.RootEntrypoint {
			return fmt.Errorf("file %s has no architecture node", mapping.Path)
		}
		if _, exists := mappings[mapping.Path]; exists {
			return fmt.Errorf("file %s is mapped more than once", mapping.Path)
		}
		if architecture.Modules[mapping.Node] == nil && !mapping.RootEntrypoint {
			return fmt.Errorf("file %s maps to non-module %s", mapping.Path, mapping.Node)
		}
		clean := filepath.ToSlash(filepath.Clean(mapping.Path))
		if mapping.Path == "" || clean == "." || filepath.IsAbs(mapping.Path) || clean == ".." || strings.HasPrefix(clean, "../") {
			return fmt.Errorf("file mapping %q must be a relative path", mapping.Path)
		}
		absolute := filepath.Join(root, filepath.FromSlash(mapping.Path))
		if !within(absolute, root) {
			return fmt.Errorf("mapped file %s is outside the Python implementation", mapping.Path)
		}
		info, statErr := os.Stat(absolute)
		if statErr != nil {
			return fmt.Errorf("mapped Python source file %s: %w", mapping.Path, statErr)
		}
		if info.IsDir() || filepath.Ext(absolute) != ".py" {
			return fmt.Errorf("mapped file %s must be a Python source file", mapping.Path)
		}
		if module := architecture.Modules[mapping.Node]; module != nil && !mapping.Explicit {
			location := moduleLocation(root, architecture, module)
			if !within(absolute, location) {
				return fmt.Errorf("mapped file %s is outside module %s folder %s", mapping.Path, mapping.Node, filepath.ToSlash(location))
			}
			fileDirectory := filepath.Dir(absolute)
			if fileDirectory != location && !withinModuleEntrypoint(fileDirectory, location, module) {
				return fmt.Errorf("mapped file %s is in an undeclared submodule folder", mapping.Path)
			}
			if mapping.EntryPoint {
				expected := filepath.Join(location, "cmd", model.ConventionalEntrypoint(mapping.EntryPointName), "main.py")
				if model.ConventionalFolder(module.Name) == "command" || model.ConventionalEntrypoint(module.Name) == model.ConventionalEntrypoint(mapping.EntryPointName) {
					expected = filepath.Join(location, "main.py")
				}
				if absolute != expected {
					return fmt.Errorf("entry point %s must follow Python convention %s", mapping.EntryPointName, filepath.ToSlash(expected))
				}
			}
		}
		mappings[mapping.Path] = mapping
	}

	return validatePythonSourceOwnership(root, architecture, mappings)
}

func validatePythonSourceOwnership(root string, architecture *model.Architecture, mappings map[string]model.FileMapping) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", ".venv", "__pycache__", "node_modules", "tmp", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".py" {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if _, mapped := mappings[relative]; mapped {
			return nil
		}
		for _, module := range architecture.Modules {
			if within(path, moduleLocation(root, architecture, module)) {
				return fmt.Errorf("Python source file %s has no architecture mapping", relative)
			}
		}
		return nil
	})
}
