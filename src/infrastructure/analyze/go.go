package analyze

import (
	"encoding/json"
	"fmt"
	"go/ast"
	goparser "go/parser"
	"go/token"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"bound/src/lib/model"
)

type goPackage struct {
	ImportPath string
	Dir        string
	Imports    []string
}

type goWorkspace struct {
	Use []struct {
		DiskPath string
	} `json:"Use"`
}

// Go checks the import graph reported by go list against the architecture.
func Go(root string, architecture *model.Architecture) error {
	if architecture.Implementation.Language != "go" {
		return fmt.Errorf("architecture implementation must use Go")
	}
	if err := validateGoFiles(root, architecture); err != nil {
		return err
	}
	moduleRoots, err := goModuleRoots(root)
	if err != nil {
		return err
	}
	packages := make([]goPackage, 0)
	seenPackages := make(map[string]bool)
	for _, moduleRoot := range moduleRoots {
		pattern := "./..."
		if relative, relativeErr := filepath.Rel(moduleRoot, root); relativeErr == nil && relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			pattern = "./" + filepath.ToSlash(relative) + "/..."
		}
		command := exec.Command("go", "list", "-json", pattern)
		command.Dir = moduleRoot
		output, err := command.Output()
		if err != nil {
			return fmt.Errorf("go list in %s: %w", filepath.ToSlash(moduleRoot), err)
		}
		decoder := json.NewDecoder(strings.NewReader(string(output)))
		for {
			var pkg goPackage
			if err := decoder.Decode(&pkg); err != nil {
				if err == io.EOF {
					break
				}
				return fmt.Errorf("decode go list output in %s: %w", filepath.ToSlash(moduleRoot), err)
			}
			key := pkg.ImportPath + "\x00" + pkg.Dir
			if !seenPackages[key] {
				packages = append(packages, pkg)
				seenPackages[key] = true
			}
		}
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

func goModuleRoots(root string) ([]string, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve Go module root: %w", err)
	}
	roots := make(map[string]bool)
	addModule := func(directory string) error {
		absolute, err := filepath.Abs(directory)
		if err != nil {
			return err
		}
		info, err := os.Stat(filepath.Join(absolute, "go.mod"))
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.IsDir() {
			return fmt.Errorf("Go module file %s is a folder", filepath.ToSlash(filepath.Join(absolute, "go.mod")))
		}
		roots[absolute] = true
		return nil
	}

	if err := addModule(root); err != nil {
		return nil, fmt.Errorf("inspect Go root: %w", err)
	}
	if len(roots) == 0 {
		for parent := filepath.Dir(root); ; parent = filepath.Dir(parent) {
			if err := addModule(parent); err != nil {
				return nil, fmt.Errorf("inspect parent Go module: %w", err)
			}
			if len(roots) > 0 || parent == filepath.Dir(parent) {
				break
			}
		}
	}
	walkErr := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root && (entry.Name() == ".git" || entry.Name() == "vendor" || entry.Name() == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Name() == "go.mod" {
			return addModule(filepath.Dir(path))
		}
		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("discover Go modules: %w", walkErr)
	}

	workspacePath := filepath.Join(root, "go.work")
	if _, statErr := os.Stat(workspacePath); statErr == nil {
		command := exec.Command("go", "work", "edit", "-json")
		command.Dir = root
		output, commandErr := command.Output()
		if commandErr != nil {
			return nil, fmt.Errorf("read Go workspace: %w", commandErr)
		}
		var workspace goWorkspace
		if err := json.Unmarshal(output, &workspace); err != nil {
			return nil, fmt.Errorf("decode Go workspace: %w", err)
		}
		for _, use := range workspace.Use {
			if err := addModule(filepath.Join(root, filepath.FromSlash(use.DiskPath))); err != nil {
				return nil, fmt.Errorf("inspect Go workspace module %s: %w", use.DiskPath, err)
			}
		}
	} else if !os.IsNotExist(statErr) {
		return nil, fmt.Errorf("inspect Go workspace: %w", statErr)
	}

	if len(roots) == 0 {
		return nil, fmt.Errorf("no go.mod found under Go implementation root %s", filepath.ToSlash(root))
	}
	result := make([]string, 0, len(roots))
	for moduleRoot := range roots {
		result = append(result, moduleRoot)
	}
	sort.Strings(result)
	return result, nil
}

func validateGoFiles(root string, architecture *model.Architecture) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve Go root: %w", err)
	}
	mappings := make(map[string]model.FileMapping, len(architecture.Files))
	// The compiler resolves the implementation locator before invoking the
	// backend. Keep all source ownership and module checks relative to that
	// resolved root rather than applying the locator a second time.
	implementationRoot := root
	for _, module := range architecture.Modules {
		location := moduleLocation(implementationRoot, architecture, module)
		info, statErr := os.Stat(location)
		if statErr != nil {
			if os.IsNotExist(statErr) && (moduleHasOnlyExplicitEntrypoints(architecture, module) || moduleHasOnlyExplicitMappings(architecture, module)) {
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
			if !mapping.Explicit {
				if !within(absolute, location) {
					return fmt.Errorf("mapped file %s is outside module %s folder %s", mapping.Path, mapping.Node, filepath.ToSlash(location))
				}
				fileDirectory := filepath.Dir(absolute)
				if fileDirectory != location && !withinModuleEntrypoint(fileDirectory, location, module) {
					return fmt.Errorf("mapped file %s is in an undeclared submodule folder", mapping.Path)
				}
			}
			if mapping.EntryPoint && !mapping.Explicit {
				expected := filepath.Join(location, "cmd", model.ConventionalEntrypoint(mapping.EntryPointName), "main.go")
				if model.ConventionalFolder(module.Name) == "command" {
					expected = filepath.Join(location, "main.go")
				} else if model.ConventionalEntrypoint(module.Name) == model.ConventionalEntrypoint(mapping.EntryPointName) {
					expected = filepath.Join(location, "main.go")
				}
				if absolute != expected {
					return fmt.Errorf("entry point %s must follow Go convention %s", mapping.EntryPointName, filepath.ToSlash(expected))
				}
			}
		}
		mappings[mapping.Path] = mapping
	}
	if err := ValidateGoQuality(root, architecture, mappings); err != nil {
		return err
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
			owned := false
			for _, module := range architecture.Modules {
				if within(path, moduleLocation(implementationRoot, architecture, module)) {
					owned = true
					break
				}
			}
			if !owned {
				return nil
			}
			return fmt.Errorf("Go source file %s has no architecture mapping", relative)
		}
		return nil
	})
}

type functionMetrics struct {
	complexity int
	maxNesting int
}

type complexityVisitor struct {
	complexity int
	depth      int
	maxDepth   int
	controls   []bool
}

// ValidateGoQuality checks configured Go quality limits for mapped files.
func ValidateGoQuality(root string, architecture *model.Architecture, mappings map[string]model.FileMapping) error {
	policy := architecture.Quality
	if policy.MaxFunctionLines == 0 && policy.MaxCyclomaticComplexity == 0 && policy.MaxNestingDepth == 0 && policy.MaxParameters == 0 && policy.MaxFileLines == 0 && !policy.Rules.OneDeclarationKindPerFile {
		return nil
	}

	paths := make([]string, 0, len(mappings))
	for path := range mappings {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, relative := range paths {
		path := filepath.Join(root, filepath.FromSlash(relative))
		fileSet := token.NewFileSet()
		file, err := goparser.ParseFile(fileSet, path, nil, 0)
		if err != nil {
			return fmt.Errorf("parse mapped Go file %s: %w", relative, err)
		}
		if policy.Rules.OneDeclarationKindPerFile {
			if err := validateDeclarationKinds(relative, file); err != nil {
				return err
			}
		}
		fileLines := fileSet.Position(file.End()).Line - fileSet.Position(file.Pos()).Line + 1
		if policy.MaxFileLines > 0 && fileLines > policy.MaxFileLines {
			return fmt.Errorf("quality violation in %s: file is %d lines, limit %d", relative, fileLines, policy.MaxFileLines)
		}
		var violation error
		ast.Inspect(file, func(node ast.Node) bool {
			if violation != nil {
				return false
			}
			function, ok := node.(*ast.FuncDecl)
			if !ok || function.Body == nil {
				return true
			}
			line := fileSet.Position(function.Pos()).Line
			name := function.Name.Name
			functionLines := fileSet.Position(function.End()).Line - line + 1
			if policy.MaxFunctionLines > 0 && functionLines > policy.MaxFunctionLines {
				violation = fmt.Errorf("quality violation in %s:%d: function %s is %d lines, limit %d", relative, line, name, functionLines, policy.MaxFunctionLines)
				return false
			}
			parameters := parameterCount(function.Type.Params)
			if policy.MaxParameters > 0 && parameters > policy.MaxParameters {
				violation = fmt.Errorf("quality violation in %s:%d: function %s has %d parameters, limit %d", relative, line, name, parameters, policy.MaxParameters)
				return false
			}
			metrics := measureFunction(function.Body)
			if policy.MaxCyclomaticComplexity > 0 && metrics.complexity > policy.MaxCyclomaticComplexity {
				violation = fmt.Errorf("quality violation in %s:%d: function %s has cyclomatic complexity %d, limit %d", relative, line, name, metrics.complexity, policy.MaxCyclomaticComplexity)
				return false
			}
			if policy.MaxNestingDepth > 0 && metrics.maxNesting > policy.MaxNestingDepth {
				violation = fmt.Errorf("quality violation in %s:%d: function %s has nesting depth %d, limit %d", relative, line, name, metrics.maxNesting, policy.MaxNestingDepth)
				return false
			}
			return false
		})
		if violation != nil {
			return violation
		}
	}
	return nil
}

func validateDeclarationKinds(relative string, file *ast.File) error {
	kinds := map[string]bool{}
	for _, declaration := range file.Decls {
		switch declaration := declaration.(type) {
		case *ast.FuncDecl:
			kinds["functions"] = true
		case *ast.GenDecl:
			switch declaration.Tok {
			case token.TYPE:
				kinds["types"] = true
			case token.CONST:
				kinds["constants"] = true
			case token.VAR:
				kinds["variables"] = true
			}
		}
	}
	if len(kinds) <= 1 {
		return nil
	}
	names := make([]string, 0, len(kinds))
	for kind := range kinds {
		names = append(names, kind)
	}
	sort.Strings(names)
	return fmt.Errorf("quality violation in %s: file mixes top-level declaration kinds (%s)", relative, strings.Join(names, ", "))
}

func parameterCount(fieldList *ast.FieldList) int {
	if fieldList == nil {
		return 0
	}
	count := 0
	for _, field := range fieldList.List {
		if len(field.Names) == 0 {
			count++
			continue
		}
		count += len(field.Names)
	}
	return count
}

func measureFunction(body *ast.BlockStmt) functionMetrics {
	visitor := &complexityVisitor{complexity: 1}
	ast.Walk(visitor, body)
	return functionMetrics{complexity: visitor.complexity, maxNesting: visitor.maxDepth}
}

func (visitor *complexityVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		last := len(visitor.controls) - 1
		if last >= 0 {
			if visitor.controls[last] {
				visitor.depth--
			}
			visitor.controls = visitor.controls[:last]
		}
		return visitor
	}

	control := isControlNode(node)
	if control {
		visitor.complexity++
		visitor.depth++
		if visitor.depth > visitor.maxDepth {
			visitor.maxDepth = visitor.depth
		}
	}
	if binary, ok := node.(*ast.BinaryExpr); ok && (binary.Op.String() == "&&" || binary.Op.String() == "||") {
		visitor.complexity++
	}
	visitor.controls = append(visitor.controls, control)
	return visitor
}

func isControlNode(node ast.Node) bool {
	switch node.(type) {
	case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt, *ast.CaseClause, *ast.CommClause:
		return true
	default:
		return false
	}
}

func moduleHasOnlyExplicitMappings(architecture *model.Architecture, module *model.Module) bool {
	prefix := module.Qualified + "."
	found := false
	for _, mapping := range architecture.Files {
		if mapping.Node != module.Qualified && !strings.HasPrefix(mapping.Node, prefix) {
			continue
		}
		found = true
		if !mapping.Explicit {
			return false
		}
	}
	return found
}

func moduleHasOnlyExplicitEntrypoints(architecture *model.Architecture, module *model.Module) bool {
	if len(module.Entrypoints) == 0 {
		return false
	}
	for entrypoint := range module.Entrypoints {
		found := false
		for _, mapping := range architecture.Files {
			if mapping.Node == module.Qualified && mapping.EntryPointName == entrypoint {
				if !mapping.Explicit {
					return false
				}
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func withinModuleEntrypoint(directory, location string, module *model.Module) bool {
	for name := range module.Entrypoints {
		if model.ConventionalFolder(module.Name) == "command" && directory == filepath.Join(location, model.ConventionalEntrypoint(name)) {
			return true
		}
		if directory == filepath.Join(location, "cmd", model.ConventionalEntrypoint(name)) {
			return true
		}
		if model.ConventionalEntrypoint(module.Name) == model.ConventionalEntrypoint(name) && directory == location {
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
		names = append(names, model.ConventionalFolder(current.Name))
	}
	for index := len(names) - 1; index >= 0; index-- {
		parts = append(parts, names[index])
	}
	return filepath.Join(append([]string{root}, parts...)...)
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
