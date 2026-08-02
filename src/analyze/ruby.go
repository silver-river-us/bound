package analyze

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bound/src/model"
)

// Ruby validates explicit source ownership for Ruby/Rails repositories.
// Rails autoloading does not expose a complete static import graph like Go's
// package tooling, so this backend focuses on path ownership and mapped-file
// existence while the Bound model carries the intended dependency graph.
func Ruby(root string, architecture *model.Architecture) error {
	if architecture.Implementation.Language != "ruby" {
		return fmt.Errorf("architecture implementation must use Ruby")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve Ruby root: %w", err)
	}
	mappings := make(map[string]bool, len(architecture.Files))
	for _, mapping := range architecture.Files {
		if mapping.Node == "" && !mapping.RootEntrypoint {
			return fmt.Errorf("file %s has no architecture node", mapping.Path)
		}
		if mappings[mapping.Path] {
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
			return fmt.Errorf("mapped file %s is outside the Ruby implementation", mapping.Path)
		}
		info, statErr := os.Stat(absolute)
		if statErr != nil {
			return fmt.Errorf("mapped source file %s: %w", mapping.Path, statErr)
		}
		if info.IsDir() {
			return fmt.Errorf("mapped source %s must be a file", mapping.Path)
		}
		mappings[mapping.Path] = true
	}
	return validateRubySourceOwnership(root, mappings)
}

func validateRubySourceOwnership(root string, mappings map[string]bool) error {
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", ".claude", "infra", "node_modules", "tmp", "log", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".rb" {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if strings.HasPrefix(relative, "api/test/") {
			return nil
		}
		if !mappings[relative] {
			return fmt.Errorf("Ruby source file %s has no architecture mapping", relative)
		}
		return nil
	})
}
