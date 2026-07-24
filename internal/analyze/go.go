package analyze

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/silver-river-us/bound/internal/model"
)

type goPackage struct {
	ImportPath string
	Dir        string
	Imports    []string
}

// Go checks the import graph reported by go list against the architecture.
func Go(root string, architecture *model.Architecture) error {
	if architecture.Implementation.Language != "go" {
		return fmt.Errorf("architecture implementation must use Go")
	}
	if err := validateGoFiles(root, architecture); err != nil {
		return err
	}
	command := exec.Command("go", "list", "-json", "./...")
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		return fmt.Errorf("go list: %w", err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(output)))
	packages := make([]goPackage, 0)
	for {
		var pkg goPackage
		if err := decoder.Decode(&pkg); err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("decode go list output: %w", err)
		}
		packages = append(packages, pkg)
	}

	owners := make(map[string]string)
	for _, pkg := range packages {
		context, err := ownerByDir(pkg.Dir, root, architecture)
		if err != nil {
			return err
		}
		if context != "" {
			owners[pkg.ImportPath] = context
		}
	}
	for _, pkg := range packages {
		from := owners[pkg.ImportPath]
		if from == "" {
			continue
		}
		for _, imported := range pkg.Imports {
			to := owners[imported]
			if to == "" || to == from || architecture.ModuleAllows(from, to) {
				continue
			}
			return fmt.Errorf("%s (%s) imports %s (%s) without a declared module dependency", pkg.ImportPath, from, imported, to)
		}
	}
	return nil
}

func validateGoFiles(root string, architecture *model.Architecture) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve Go root: %w", err)
	}
	mappings := make(map[string]model.FileMapping, len(architecture.Files))
	implementationRoot := filepath.Join(root, filepath.FromSlash(architecture.Implementation.Locator))
	for _, module := range architecture.Modules {
		location := moduleLocation(implementationRoot, architecture, module)
		info, statErr := os.Stat(location)
		if statErr != nil {
			return fmt.Errorf("module %s folder %s: %w", module.Qualified, filepath.ToSlash(location), statErr)
		}
		if !info.IsDir() {
			return fmt.Errorf("module %s path %s must be a folder", module.Qualified, filepath.ToSlash(location))
		}
	}
	for _, mapping := range architecture.Files {
		if mapping.Node == "" {
			return fmt.Errorf("file %s has no architecture node", mapping.Path)
		}
		if _, exists := mappings[mapping.Path]; exists {
			return fmt.Errorf("file %s is mapped more than once", mapping.Path)
		}
		if architecture.Modules[mapping.Node] == nil {
			return fmt.Errorf("file %s maps to non-module %s", mapping.Path, mapping.Node)
		}
		absolute := filepath.Join(root, filepath.FromSlash(mapping.Path))
		info, statErr := os.Stat(absolute)
		if statErr != nil {
			return fmt.Errorf("mapped Go file %s: %w", mapping.Path, statErr)
		}
		if info.IsDir() || filepath.Ext(absolute) != ".go" {
			return fmt.Errorf("mapped file %s must be a Go source file", mapping.Path)
		}
		if !within(absolute, implementationRoot) {
			return fmt.Errorf("mapped file %s is outside the architecture implementation", mapping.Path)
		}
		if module := architecture.Modules[mapping.Node]; module != nil {
			location := moduleLocation(implementationRoot, architecture, module)
			if !within(absolute, location) {
				return fmt.Errorf("mapped file %s is outside module %s folder %s", mapping.Path, mapping.Node, filepath.ToSlash(location))
			}
			fileDirectory := filepath.Dir(absolute)
			if fileDirectory != location && !withinModuleEntrypoint(fileDirectory, location, module) {
				return fmt.Errorf("mapped file %s is in an undeclared submodule folder", mapping.Path)
			}
			if mapping.EntryPoint {
				expected := filepath.Join(location, "cmd", kebabCase(mapping.EntryPointName), "main.go")
				if kebabCase(module.Name) == kebabCase(mapping.EntryPointName) {
					expected = filepath.Join(location, "main.go")
				}
				if absolute != expected {
					return fmt.Errorf("entry point %s must follow Go convention %s", mapping.EntryPointName, filepath.ToSlash(expected))
				}
			}
		}
		mappings[mapping.Path] = mapping
	}
	return filepath.WalkDir(implementationRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if _, ok := mappings[relative]; !ok {
			return fmt.Errorf("Go source file %s has no architecture mapping", relative)
		}
		return nil
	})
}

func withinModuleEntrypoint(directory, location string, module *model.Module) bool {
	for name := range module.Entrypoints {
		if directory == filepath.Join(location, "cmd", kebabCase(name)) {
			return true
		}
		if kebabCase(module.Name) == kebabCase(name) && directory == location {
			return true
		}
	}
	return false
}

func contextForNode(architecture *model.Architecture, node string) *model.Context {
	if module := architecture.Modules[node]; module != nil {
		return architecture.Contexts[module.Context]
	}
	if context := architecture.Contexts[node]; context != nil {
		return context
	}
	for _, context := range architecture.Contexts {
		if _, ok := context.Interfaces[node]; ok {
			return context
		}
	}
	return nil
}

func moduleLocation(root string, architecture *model.Architecture, module *model.Module) string {
	var parts []string
	var names []string
	for current := module; current != nil; current = architecture.Modules[current.Parent] {
		names = append(names, snakeCase(current.Name))
	}
	for index := len(names) - 1; index >= 0; index-- {
		parts = append(parts, names[index])
	}
	return filepath.Join(append([]string{root}, parts...)...)
}

func snakeCase(value string) string {
	var result []rune
	for index, character := range value {
		if index > 0 && character >= 'A' && character <= 'Z' {
			result = append(result, '_')
		}
		if character >= 'A' && character <= 'Z' {
			character += 'a' - 'A'
		}
		result = append(result, character)
	}
	return string(result)
}

func kebabCase(value string) string {
	return strings.ReplaceAll(snakeCase(value), "_", "-")
}

func within(file, directory string) bool {
	absoluteFile, fileErr := filepath.Abs(file)
	absoluteDirectory, directoryErr := filepath.Abs(directory)
	if fileErr != nil || directoryErr != nil {
		return false
	}
	return absoluteFile == absoluteDirectory || strings.HasPrefix(absoluteFile, absoluteDirectory+string(filepath.Separator))
}

func ownerByDir(dir, root string, architecture *model.Architecture) (string, error) {
	absoluteDir, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	owner := ""
	for _, mapping := range architecture.Files {
		mappedDir, err := filepath.Abs(filepath.Join(root, filepath.Dir(filepath.FromSlash(mapping.Path))))
		if err != nil || mappedDir != absoluteDir {
			continue
		}
		module := architecture.Modules[mapping.Node]
		if module == nil {
			continue
		}
		if owner != "" && owner != module.Qualified {
			return "", fmt.Errorf("Go package %s maps to multiple modules: %s and %s", dir, owner, module.Qualified)
		}
		owner = module.Qualified
	}
	return owner, nil
}
