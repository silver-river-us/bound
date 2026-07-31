package parser

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"bound/internal/model"
)

var (
	architectureRE   = regexp.MustCompile(`^architecture\s+([A-Za-z_][A-Za-z0-9_]*)\s+do$`)
	domainRE         = regexp.MustCompile(`^(entity|value)\s+([A-Za-z_][A-Za-z0-9_]*)\s+do$`)
	moduleRE         = regexp.MustCompile(`^module\s+([A-Za-z_][A-Za-z0-9_]*)\s+do$`)
	rootEntrypointRE = regexp.MustCompile(`^entrypoint\s+:([A-Za-z_][A-Za-z0-9_]*)$`)
	qualityRE        = regexp.MustCompile(`^quality\s+do$`)
	qualityRuleRE    = regexp.MustCompile(`^(max_function_lines|max_cyclomatic_complexity|max_nesting_depth|max_parameters|max_file_lines)\s+([0-9]+)$`)
	qualityRulesRE   = regexp.MustCompile(`^rules\s+do$`)
	implementsRE     = regexp.MustCompile(`^implements\s+([A-Za-z_][A-Za-z0-9_.]*)$`)
	usesRE           = regexp.MustCompile(`^uses\s+([A-Za-z_][A-Za-z0-9_.]*)$`)
	entrypointRE     = regexp.MustCompile(`^entrypoint\s+([A-Za-z_][A-Za-z0-9_]*)(?:\s+"([^"]+)")?$`)
	fileRE           = regexp.MustCompile(`^file\s+:([A-Za-z_][A-Za-z0-9_]*)$`)
	filesRE          = regexp.MustCompile(`^files\s+\[([^]]*)\]$`)
	stateRE          = regexp.MustCompile(`^state\s+:([A-Za-z_][A-Za-z0-9_]*)\s+:([A-Za-z_][A-Za-z0-9_.]*(?:\[\])?)$`)
	importRE         = regexp.MustCompile(`^import\s+([A-Za-z_][A-Za-z0-9_]*)\s+from\s+"([^"]*)"$`)
	contextRE        = regexp.MustCompile(`^context\s+([A-Za-z_][A-Za-z0-9_]*)\s+do$`)
	implementationRE = regexp.MustCompile(`^implementation\s+([A-Za-z_][A-Za-z0-9_+-]*)\s+"([^"]+)"$`)
	exposesRE        = regexp.MustCompile(`^exposes\s+([A-Za-z_][A-Za-z0-9_]*)$`)
	interfaceRE      = regexp.MustCompile(`^interface\s+([A-Za-z_][A-Za-z0-9_]*)\s+do$`)
	behaviorRE       = regexp.MustCompile(`^behavior\s+([A-Za-z_][A-Za-z0-9_]*)\(([^)]*)\)(?:\s+returns\s+([A-Za-z_][A-Za-z0-9_.]*(?:\[\])?))?$`)
	relationRE       = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*->\s*([A-Za-z_][A-Za-z0-9_]*)(?:\s+via\s+([A-Za-z_][A-Za-z0-9_]*))?$`)
)

func Parse(r io.Reader) (*model.Architecture, error) {
	scanner := bufio.NewScanner(r)
	var architecture *model.Architecture
	var current *model.Context
	var currentInterface *model.Interface
	var currentObject *model.Object
	var moduleStack []*model.Module
	insideQuality := false
	insideQualityRules := false
	pendingDescription := ""
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(strings.SplitN(scanner.Text(), "#", 2)[0])
		if line == "" {
			continue
		}
		if line == `"""` {
			description, err := scanDescription(scanner, &lineNumber)
			if err != nil {
				return nil, err
			}
			pendingDescription = description
			continue
		}
		if insideQualityRules {
			if line == "end" {
				insideQualityRules = false
				continue
			}
			if line != "one_declaration_kind_per_file" {
				return nil, syntaxError(lineNumber, "expected quality rule or end")
			}
			architecture.Quality.Rules.OneDeclarationKindPerFile = true
			continue
		}
		if insideQuality {
			if line == "end" {
				insideQuality = false
				continue
			}
			if qualityRulesRE.MatchString(line) {
				insideQualityRules = true
				continue
			}
			match := qualityRuleRE.FindStringSubmatch(line)
			if match == nil {
				return nil, syntaxError(lineNumber, "expected quality rule or end")
			}
			value, err := strconv.Atoi(match[2])
			if err != nil || value < 1 {
				return nil, syntaxError(lineNumber, "quality limits must be positive integers")
			}
			switch match[1] {
			case "max_function_lines":
				architecture.Quality.MaxFunctionLines = value
			case "max_cyclomatic_complexity":
				architecture.Quality.MaxCyclomaticComplexity = value
			case "max_nesting_depth":
				architecture.Quality.MaxNestingDepth = value
			case "max_parameters":
				architecture.Quality.MaxParameters = value
			case "max_file_lines":
				architecture.Quality.MaxFileLines = value
			}
			continue
		}
		if architecture == nil {
			match := architectureRE.FindStringSubmatch(line)
			if match == nil {
				return nil, syntaxError(lineNumber, "expected architecture declaration")
			}
			architecture = &model.Architecture{Name: match[1], Description: pendingDescription, Contexts: map[string]*model.Context{}, Objects: map[string]*model.Object{}, Modules: map[string]*model.Module{}}
			pendingDescription = ""
			continue
		}
		if currentObject != nil {
			if line == "end" {
				currentObject = nil
				continue
			}
			if match := behaviorRE.FindStringSubmatch(line); match != nil {
				if currentObject.Kind != "entity" {
					return nil, syntaxError(lineNumber, "values cannot declare behavior")
				}
				if _, exists := currentObject.Operations[match[1]]; exists {
					return nil, syntaxError(lineNumber, "duplicate behavior")
				}
				operation, err := parseOperation(match, line)
				if err != nil {
					return nil, syntaxError(lineNumber, err.Error())
				}
				currentObject.Operations[match[1]] = operation
				pendingDescription = ""
				continue
			}
			match := stateRE.FindStringSubmatch(line)
			if match == nil {
				return nil, syntaxError(lineNumber, "expected state or end")
			}
			if _, exists := currentObject.Attributes[match[1]]; exists {
				return nil, syntaxError(lineNumber, "duplicate state")
			}
			currentObject.Attributes[match[1]] = model.Attribute{Name: match[1], Type: match[2], Description: pendingDescription}
			pendingDescription = ""
			continue
		}
		if current == nil && line == "end" {
			return architecture, nil
		}
		if current == nil {
			if architecture.Implementation.Language == "" {
				match := implementationRE.FindStringSubmatch(line)
				if match == nil {
					return nil, syntaxError(lineNumber, "expected architecture implementation")
				}
				architecture.Implementation.Language = match[1]
				architecture.Implementation.Locator = match[2]
				continue
			}
			if match := implementationRE.FindStringSubmatch(line); match != nil {
				return nil, syntaxError(lineNumber, "duplicate implementation")
			}
			if match := rootEntrypointRE.FindStringSubmatch(line); match != nil {
				architecture.Files = append(architecture.Files, model.FileMapping{
					EntryPoint:     true,
					EntryPointName: match[1],
					Explicit:       true,
					RootEntrypoint: true,
				})
				continue
			}
			if qualityRE.MatchString(line) {
				insideQuality = true
				continue
			}
			if match := importRE.FindStringSubmatch(line); match != nil {
				if err := validateImportPath(match[2]); err != nil {
					return nil, syntaxError(lineNumber, err.Error())
				}
				kind, err := importKind(match[2])
				if err != nil {
					return nil, syntaxError(lineNumber, err.Error())
				}
				architecture.Imports = append(architecture.Imports, model.Import{Symbol: match[1], Kind: kind, Path: match[2]})
				continue
			}
			if match := domainRE.FindStringSubmatch(line); match != nil {
				if _, exists := architecture.Objects[match[2]]; exists {
					return nil, syntaxError(lineNumber, "duplicate object")
				}
				currentObject = &model.Object{Name: match[2], Kind: match[1], Description: pendingDescription, Attributes: map[string]model.Attribute{}, Operations: map[string]model.Operation{}}
				pendingDescription = ""
				architecture.Objects[match[2]] = currentObject
				continue
			}
			if match := contextRE.FindStringSubmatch(line); match != nil {
				if _, exists := architecture.Contexts[match[1]]; exists {
					return nil, syntaxError(lineNumber, "duplicate context")
				}
				current = &model.Context{Name: match[1], Description: pendingDescription, Exposes: map[string]bool{}, Interfaces: map[string]*model.Interface{}, Modules: map[string]*model.Module{}}
				pendingDescription = ""
				architecture.Contexts[current.Name] = current
				continue
			}
			if match := relationRE.FindStringSubmatch(line); match != nil {
				architecture.Relations = append(architecture.Relations, model.Relation{From: match[1], To: match[2], Via: match[3], Description: pendingDescription})
				pendingDescription = ""
				continue
			}
			return nil, syntaxError(lineNumber, "expected context, relationship, or end")
		}
		if currentInterface != nil {
			if line == "end" {
				currentInterface = nil
				continue
			}
			if match := domainRE.FindStringSubmatch(line); match != nil {
				if _, exists := currentInterface.Types[match[2]]; exists {
					return nil, syntaxError(lineNumber, "duplicate interface type")
				}
				currentObject = &model.Object{Name: match[2], Kind: match[1], Description: pendingDescription, Attributes: map[string]model.Attribute{}, Operations: map[string]model.Operation{}}
				pendingDescription = ""
				currentInterface.Types[match[2]] = currentObject
				continue
			}
			match := behaviorRE.FindStringSubmatch(line)
			if match == nil {
				return nil, syntaxError(lineNumber, "expected entity, value, behavior, or end")
			}
			if _, exists := currentInterface.Operations[match[1]]; exists {
				return nil, syntaxError(lineNumber, "duplicate behavior")
			}
			parameters, err := parseParameters(match[2])
			if err != nil {
				return nil, syntaxError(lineNumber, err.Error())
			}
			currentInterface.Operations[match[1]] = model.Operation{
				Name:        match[1],
				Signature:   strings.TrimSpace(strings.TrimPrefix(line, "behavior "+match[1])),
				Description: pendingDescription,
				Parameters:  parameters,
				Returns:     match[3],
			}
			pendingDescription = ""
			continue
		}
		if len(moduleStack) > 0 {
			module := moduleStack[len(moduleStack)-1]
			if line == "end" {
				moduleStack = moduleStack[:len(moduleStack)-1]
				continue
			}
			if match := moduleRE.FindStringSubmatch(line); match != nil {
				child, err := addModule(architecture, current, module, match[1])
				if err != nil {
					return nil, syntaxError(lineNumber, err.Error())
				}
				moduleStack = append(moduleStack, child)
				continue
			}
			if match := implementsRE.FindStringSubmatch(line); match != nil {
				if module.Implements != "" {
					return nil, syntaxError(lineNumber, "duplicate implements")
				}
				module.Implements = match[1]
				continue
			}
			if match := usesRE.FindStringSubmatch(line); match != nil {
				module.Uses[match[1]] = true
				continue
			}
			if match := fileRE.FindStringSubmatch(line); match != nil {
				module.Files = append(module.Files, match[1])
				continue
			}
			if match := filesRE.FindStringSubmatch(line); match != nil {
				files, err := parseFileAtoms(match[1])
				if err != nil {
					return nil, syntaxError(lineNumber, err.Error())
				}
				module.Files = append(module.Files, files...)
				continue
			}
			if match := entrypointRE.FindStringSubmatch(line); match != nil {
				module.Entrypoints[match[1]] = true
				if match[2] != "" {
					architecture.Files = append(architecture.Files, model.FileMapping{
						Path:           match[2],
						Node:           module.Qualified,
						EntryPoint:     true,
						EntryPointName: match[1],
						Explicit:       true,
					})
				}
				continue
			}
			return nil, syntaxError(lineNumber, "expected module, implements, uses, file, entrypoint, or end")
		}
		if line == "end" {
			current = nil
			continue
		}
		switch {
		case importRE.MatchString(line):
			match := importRE.FindStringSubmatch(line)
			if err := validateImportPath(match[2]); err != nil {
				return nil, syntaxError(lineNumber, err.Error())
			}
			kind, err := importKind(match[2])
			if err != nil {
				return nil, syntaxError(lineNumber, err.Error())
			}
			current.Imports = append(current.Imports, model.Import{Symbol: match[1], Kind: kind, Path: match[2]})
		case exposesRE.MatchString(line):
			current.Exposes[exposesRE.FindStringSubmatch(line)[1]] = true
		case interfaceRE.MatchString(line):
			match := interfaceRE.FindStringSubmatch(line)
			if _, exists := current.Interfaces[match[1]]; exists {
				return nil, syntaxError(lineNumber, "duplicate interface")
			}
			currentInterface = &model.Interface{Name: match[1], Description: pendingDescription, Types: map[string]*model.Object{}, Operations: map[string]model.Operation{}}
			pendingDescription = ""
			current.Interfaces[match[1]] = currentInterface
		case moduleRE.MatchString(line):
			match := moduleRE.FindStringSubmatch(line)
			module, err := addModule(architecture, current, nil, match[1])
			if err != nil {
				return nil, syntaxError(lineNumber, err.Error())
			}
			moduleStack = append(moduleStack, module)
		default:
			return nil, syntaxError(lineNumber, "expected import, interface, module, exposes, or end")
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if architecture == nil {
		return nil, fmt.Errorf("empty Bound document")
	}
	if insideQualityRules {
		return nil, fmt.Errorf("line %d: unclosed quality rules", lineNumber)
	}
	if insideQuality {
		return nil, fmt.Errorf("line %d: unclosed quality policy", lineNumber)
	}
	if currentObject != nil {
		return nil, fmt.Errorf("line %d: unclosed domain type %s", lineNumber, currentObject.Name)
	}
	if len(moduleStack) > 0 {
		return nil, fmt.Errorf("line %d: unclosed module %s", lineNumber, moduleStack[len(moduleStack)-1].Qualified)
	}
	if current != nil {
		if currentInterface != nil {
			return nil, fmt.Errorf("line %d: unclosed interface %s", lineNumber, currentInterface.Name)
		}
		return nil, fmt.Errorf("line %d: unclosed context %s", lineNumber, current.Name)
	}
	return nil, fmt.Errorf("unclosed architecture %s", architecture.Name)
}

func parseParameters(source string) ([]model.Parameter, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil, nil
	}
	parts := strings.Split(source, ",")
	parameters := make([]model.Parameter, 0, len(parts))
	names := map[string]bool{}
	for _, part := range parts {
		fields := strings.Fields(strings.TrimSpace(part))
		if len(fields) != 2 || !identifierRE(fields[0]) || !typeNameRE(fields[1]) {
			return nil, fmt.Errorf("invalid behavior parameter %q", strings.TrimSpace(part))
		}
		if names[fields[0]] {
			return nil, fmt.Errorf("duplicate behavior parameter %s", fields[0])
		}
		names[fields[0]] = true
		parameters = append(parameters, model.Parameter{Name: fields[0], Type: fields[1]})
	}
	return parameters, nil
}

func parseFileAtoms(source string) ([]string, error) {
	parts := strings.Split(source, ",")
	files := make([]string, 0, len(parts))
	for _, part := range parts {
		atom := strings.TrimSpace(part)
		if len(atom) < 2 || atom[0] != ':' || !identifierRE(atom[1:]) {
			return nil, fmt.Errorf("invalid file atom %q", atom)
		}
		files = append(files, atom[1:])
	}
	return files, nil
}

func importKind(importPath string) (string, error) {
	switch filepath.Ext(importPath) {
	case ".bo":
		return "contracts", nil
	case ".bom":
		return "map", nil
	default:
		return "", fmt.Errorf("import %q must target a .bo or .bom file", importPath)
	}
}

func validateImportPath(importPath string) error {
	if importPath == "" {
		return fmt.Errorf("import path cannot be empty")
	}
	if filepath.IsAbs(importPath) {
		return fmt.Errorf("import path %q must be relative", importPath)
	}
	clean := filepath.Clean(importPath)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("import path %q must stay within the source tree", importPath)
	}
	return nil
}

func parseOperation(match []string, line string) (model.Operation, error) {
	parameters, err := parseParameters(match[2])
	if err != nil {
		return model.Operation{}, err
	}
	return model.Operation{
		Name:       match[1],
		Signature:  strings.TrimSpace(strings.TrimPrefix(line, "behavior "+match[1])),
		Parameters: parameters,
		Returns:    match[3],
	}, nil
}

func identifierRE(value string) bool {
	return regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString(value)
}

func typeNameRE(value string) bool {
	return regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_.]*(?:\[\])?$`).MatchString(value)
}

func addModule(architecture *model.Architecture, context *model.Context, parent *model.Module, name string) (*model.Module, error) {
	qualified := context.Name + "." + name
	if name == context.Name {
		qualified = context.Name
	}
	children := context.Modules
	parentName := ""
	if parent != nil {
		qualified = parent.Qualified + "." + name
		children = parent.Modules
		parentName = parent.Qualified
	}
	if _, exists := children[name]; exists {
		return nil, fmt.Errorf("duplicate module")
	}
	for sibling := range children {
		if model.ConventionalFolder(sibling) == model.ConventionalFolder(name) {
			return nil, fmt.Errorf("module folder name collides with %s", sibling)
		}
	}
	if _, exists := architecture.Modules[qualified]; exists {
		return nil, fmt.Errorf("duplicate qualified module %s", qualified)
	}
	module := &model.Module{
		Name:        name,
		Qualified:   qualified,
		Context:     context.Name,
		Parent:      parentName,
		Uses:        map[string]bool{},
		Modules:     map[string]*model.Module{},
		Entrypoints: map[string]bool{},
	}
	children[name] = module
	architecture.Modules[qualified] = module
	return module, nil
}

// ParseMap parses a .bom source map for an architecture.
func ParseMap(r io.Reader) (string, []model.FileMapping, error) {
	scanner := bufio.NewScanner(r)
	mapRE := regexp.MustCompile(`^map\s+([A-Za-z_][A-Za-z0-9_]*)\s+do$`)
	fileRE := regexp.MustCompile(`^"([^"]+)"\s*->\s*([A-Za-z_][A-Za-z0-9_.]*)$`)
	entrypointFileRE := regexp.MustCompile(`^entrypoint\s+([A-Za-z_][A-Za-z0-9_]*)\s+"([^"]+)"\s*->\s*([A-Za-z_][A-Za-z0-9_.]*)$`)
	name := ""
	files := make([]model.FileMapping, 0)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(strings.SplitN(scanner.Text(), "#", 2)[0])
		if line == "" {
			continue
		}
		if name == "" {
			match := mapRE.FindStringSubmatch(line)
			if match == nil {
				return "", nil, syntaxError(lineNumber, "expected map declaration")
			}
			name = match[1]
			continue
		}
		if line == "end" {
			return name, files, nil
		}
		if match := fileRE.FindStringSubmatch(line); match != nil {
			files = append(files, model.FileMapping{Path: match[1], Node: match[2]})
			continue
		}
		if match := entrypointFileRE.FindStringSubmatch(line); match != nil {
			files = append(files, model.FileMapping{Path: match[2], Node: match[3], EntryPoint: true, EntryPointName: match[1], Explicit: true})
			continue
		}
		return "", nil, syntaxError(lineNumber, "expected file mapping, entrypoint, or end")
	}
	if err := scanner.Err(); err != nil {
		return "", nil, err
	}
	if name == "" {
		return "", nil, fmt.Errorf("empty Bound map")
	}
	return "", nil, fmt.Errorf("unclosed map %s", name)
}

// ParseFile parses an architecture and its optional relative imports.
func ParseFile(path string) (*model.Architecture, error) {
	return parseFile(path, map[string]bool{})
}

func parseFile(path string, visiting map[string]bool) (*model.Architecture, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if visiting[absPath] {
		return nil, fmt.Errorf("cyclic Bound import: %s", path)
	}
	visiting[absPath] = true
	defer delete(visiting, absPath)
	file, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	architecture, err := Parse(file)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	for _, declaration := range architecture.Imports {
		importPath := declaration.Path
		importedPath := filepath.Join(filepath.Dir(absPath), importPath)
		if declaration.Kind == "map" {
			mapFile, err := os.Open(importedPath)
			if err != nil {
				return nil, fmt.Errorf("open import %s: %w", importPath, err)
			}
			mapName, files, parseErr := ParseMap(mapFile)
			mapFile.Close()
			if parseErr != nil {
				return nil, fmt.Errorf("parse import %s: %w", importPath, parseErr)
			}
			if mapName != declaration.Symbol {
				return nil, fmt.Errorf("map %s defines %s, imported as %s", importPath, mapName, declaration.Symbol)
			}
			architecture.Files = append(architecture.Files, files...)
			continue
		}
		imported, err := parseFile(importedPath, visiting)
		if err != nil {
			return nil, err
		}
		if imported.Name != declaration.Symbol {
			return nil, fmt.Errorf("architecture %s defines %s, imported as %s", importPath, imported.Name, declaration.Symbol)
		}
		if err := merge(architecture, imported); err != nil {
			return nil, fmt.Errorf("import %s: %w", importPath, err)
		}
	}
	fragmentVisiting := map[string]bool{}
	for _, context := range architecture.Contexts {
		for _, imported := range context.Imports {
			if imported.Kind != "contracts" {
				return nil, fmt.Errorf("context %s can only import .bo contract fragments, got %s", context.Name, imported.Path)
			}
			importPath := imported.Path
			importedPath := filepath.Join(filepath.Dir(absPath), importPath)
			if err := loadContextFragment(importedPath, imported.Symbol, context, fragmentVisiting); err != nil {
				return nil, fmt.Errorf("context %s import %s: %w", context.Name, importPath, err)
			}
		}
		context.Imports = nil
	}
	return architecture, nil
}

func loadContextFragment(path, contractName string, target *model.Context, visiting map[string]bool) error {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if visiting[absolute] {
		return fmt.Errorf("cyclic Bound fragment import: %s", path)
	}
	visiting[absolute] = true
	defer delete(visiting, absolute)

	content, err := os.ReadFile(absolute)
	if err != nil {
		return err
	}
	wrapped := "architecture Fragment do\nimplementation fragment \"./\"\ncontext FragmentContext do\n" +
		string(content) + "\nend\nend\n"
	fragment, err := Parse(strings.NewReader(wrapped))
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	source := fragment.Contexts["FragmentContext"]
	if len(source.Modules) > 0 || len(source.Exposes) > 0 {
		return fmt.Errorf("fragment %s may only contain imports and interfaces", path)
	}
	contract, exists := source.Interfaces[contractName]
	if !exists {
		return fmt.Errorf("fragment %s does not define interface %s", path, contractName)
	}
	if _, exists := target.Interfaces[contractName]; exists {
		return fmt.Errorf("duplicate interface %s", contractName)
	}
	target.Interfaces[contractName] = contract
	for _, imported := range source.Imports {
		if imported.Kind != "contracts" {
			return fmt.Errorf("fragment %s can only import .bo contract fragments, got %s", path, imported.Path)
		}
		if err := loadContextFragment(filepath.Join(filepath.Dir(absolute), imported.Path), imported.Symbol, target, visiting); err != nil {
			return fmt.Errorf("import %s: %w", imported.Path, err)
		}
	}
	return nil
}

func merge(target, imported *model.Architecture) error {
	for name, context := range imported.Contexts {
		if _, exists := target.Contexts[name]; exists {
			return fmt.Errorf("duplicate context %s", name)
		}
		target.Contexts[name] = context
	}
	for name, object := range imported.Objects {
		if _, exists := target.Objects[name]; exists {
			return fmt.Errorf("duplicate object %s", name)
		}
		target.Objects[name] = object
	}
	for name, module := range imported.Modules {
		if _, exists := target.Modules[name]; exists {
			return fmt.Errorf("duplicate module %s", name)
		}
		target.Modules[name] = module
	}
	target.Relations = append(target.Relations, imported.Relations...)
	target.Files = append(target.Files, imported.Files...)
	return nil
}

func syntaxError(line int, message string) error { return fmt.Errorf("line %d: %s", line, message) }

func scanDescription(scanner *bufio.Scanner, lineNumber *int) (string, error) {
	lines := make([]string, 0)
	for scanner.Scan() {
		*lineNumber++
		line := scanner.Text()
		if strings.TrimSpace(line) == `"""` {
			return strings.TrimSpace(strings.Join(lines, "\n")), nil
		}
		lines = append(lines, strings.TrimSpace(line))
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", fmt.Errorf("line %d: unclosed documentation block", *lineNumber)
}
