package analyze

import (
	"encoding/json"
	"fmt"
	"io"
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
		if context := ownerByDir(pkg.Dir, root, architecture); context != "" {
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
			if to == "" || to == from || architecture.Allows(from, to) {
				continue
			}
			return fmt.Errorf("%s (%s) imports %s (%s) without a declared relationship", pkg.ImportPath, from, imported, to)
		}
	}
	return nil
}

func ownerByDir(dir, root string, architecture *model.Architecture) string {
	absoluteDir, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	for name, context := range architecture.Contexts {
		if context.Implementation.Language != "go" {
			continue
		}
		location, err := filepath.Abs(filepath.Join(root, context.Implementation.Locator))
		if err != nil {
			continue
		}
		if absoluteDir == location || strings.HasPrefix(absoluteDir, location+string(filepath.Separator)) {
			return name
		}
	}
	return ""
}
