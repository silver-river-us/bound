package analyze

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"

	"github.com/silver-river-us/bound/internal/model"
)

type goPackage struct {
	ImportPath string
	Imports    []string
}

// Go checks the import graph reported by go list against the architecture.
func Go(root string, architecture *model.Architecture) error {
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

	paths := make([]string, 0, len(architecture.Contexts))
	for _, context := range architecture.Contexts {
		if context.Implementation.Language == "go" {
			paths = append(paths, context.Implementation.Locator)
		}
	}
	sort.Strings(paths)
	for _, pkg := range packages {
		from := owner(pkg.ImportPath, architecture, paths)
		if from == "" {
			continue
		}
		for _, imported := range pkg.Imports {
			to := owner(imported, architecture, paths)
			if to == "" || to == from || architecture.Allows(from, to) {
				continue
			}
			return fmt.Errorf("%s (%s) imports %s (%s) without a declared relationship", pkg.ImportPath, from, imported, to)
		}
	}
	return nil
}

func owner(importPath string, architecture *model.Architecture, paths []string) string {
	for _, path := range paths {
		if importPath == path || strings.HasPrefix(importPath, path+"/") {
			for name, context := range architecture.Contexts {
				if context.Implementation.Locator == path {
					return name
				}
			}
		}
	}
	return ""
}
