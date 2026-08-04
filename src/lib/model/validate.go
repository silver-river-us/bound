package model

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

func validationError(span Span, message, suggestion string) error {
	return &Error{Message: message, Span: span, Suggestion: suggestion}
}

func suggestion(kind, name string, values interface{}) string {
	// Keep suggestions deterministic and conservative: only suggest a close name.
	var candidates []string
	switch typed := values.(type) {
	case map[string]*Context:
		for value := range typed {
			candidates = append(candidates, value)
		}
	case map[string]*Interface:
		for value := range typed {
			candidates = append(candidates, value)
		}
	}
	best := ""
	bestDistance := len(name)
	for _, candidate := range candidates {
		distance := editDistance(strings.ToLower(name), strings.ToLower(candidate))
		if distance < bestDistance {
			best, bestDistance = candidate, distance
		}
	}
	if best != "" && bestDistance <= 2 {
		return fmt.Sprintf("did you mean %s %q?", kind, best)
	}
	return ""
}

func editDistance(left, right string) int {
	row := make([]int, len(right)+1)
	for index := range row {
		row[index] = index
	}
	for i, leftChar := range left {
		next := make([]int, len(right)+1)
		next[0] = i + 1
		for j, rightChar := range right {
			cost := 0
			if leftChar != rightChar {
				cost = 1
			}
			next[j+1] = min(min(next[j]+1, row[j+1]+1), row[j]+cost)
		}
		row = next
	}
	return row[len(right)]
}

func (a *Architecture) Validate() error {
	if a.Name == "" {
		return fmt.Errorf("architecture name is required")
	}
	if a.Implementation.Language == "" || a.Implementation.Locator == "" {
		return fmt.Errorf("architecture must declare an implementation")
	}
	if err := a.materializeModuleFiles(); err != nil {
		return err
	}
	for name, module := range a.Modules {
		if module.Qualified != name || module.Name == "" || module.Context == "" {
			return fmt.Errorf("architecture has an invalid module")
		}
		context := a.Contexts[module.Context]
		if context == nil {
			return validationError(module.Span, fmt.Sprintf("module %s references unknown context %s", name, module.Context), suggestion("context", module.Context, a.Contexts))
		}
		if module.Implements != "" {
			if _, ok := context.Interfaces[module.Implements]; !ok {
				return validationError(module.Span, fmt.Sprintf("module %s implements unknown interface %s", name, module.Implements), suggestion("interface", module.Implements, context.Interfaces))
			}
		}
		for dependency := range module.Uses {
			if !a.validModuleDependency(module, dependency) {
				return validationError(module.Span, fmt.Sprintf("module %s uses unknown interface or module %s", name, dependency), "declare the interface or module before using it")
			}
		}
	}
	files := make(map[string]FileMapping, len(a.Files))
	for _, file := range a.Files {
		clean := filepath.ToSlash(filepath.Clean(file.Path))
		if file.Path == "" || clean == "." || filepath.IsAbs(file.Path) || clean == ".." || strings.HasPrefix(clean, "../") {
			return fmt.Errorf("file mapping %q must be a relative path", file.Path)
		}
		if file.Path != clean {
			return fmt.Errorf("file mapping %q must use a normalized relative path", file.Path)
		}
		if _, exists := files[file.Path]; exists {
			return fmt.Errorf("file %s is mapped more than once", file.Path)
		}
		if a.Modules[file.Node] == nil && !(file.RootEntrypoint && file.Node == "") {
			return fmt.Errorf("file %s must map to a private module, got %s", file.Path, file.Node)
		}
		files[file.Path] = file
	}
	mappedEntrypoints := map[string]int{}
	for _, file := range a.Files {
		if !file.EntryPoint {
			continue
		}
		mapping := files[file.Path]
		if mapping.Node == "" {
			if mapping.RootEntrypoint {
				continue
			}
			return fmt.Errorf("entry point %s must be mapped to a module", file.Path)
		}
		module := a.Modules[mapping.Node]
		if module == nil || !module.Entrypoints[file.EntryPointName] {
			return fmt.Errorf("entry point %s maps undeclared name %s on module %s", file.Path, file.EntryPointName, file.Node)
		}
		key := file.Node + "." + file.EntryPointName
		mappedEntrypoints[key]++
		if mappedEntrypoints[key] > 1 {
			return fmt.Errorf("entry point %s on module %s is mapped more than once", file.EntryPointName, file.Node)
		}
	}
	for name, module := range a.Modules {
		for entrypoint := range module.Entrypoints {
			if mappedEntrypoints[name+"."+entrypoint] != 1 {
				return fmt.Errorf("entry point %s on module %s must have exactly one source mapping", entrypoint, name)
			}
		}
	}
	for name, object := range a.Objects {
		if object.Name == "" || object.Name != name || (object.Kind != "entity" && object.Kind != "value") {
			return fmt.Errorf("architecture has an invalid domain type")
		}
		if err := validateObject(name, object); err != nil {
			return fmt.Errorf("architecture domain type %s: %w", name, err)
		}
		if err := validateObjectOperations(name, object, a.validArchitectureType); err != nil {
			return fmt.Errorf("architecture domain type %s: %w", name, err)
		}
	}
	for name, context := range a.Contexts {
		for interfaceName, contract := range context.Interfaces {
			if interfaceName != contract.Name || contract.Name == "" {
				return fmt.Errorf("context %s has an invalid interface", name)
			}
			for typeName, object := range contract.Types {
				if err := validateObject(typeName, object); err != nil {
					return fmt.Errorf("interface %s.%s: %w", name, interfaceName, err)
				}
				for _, attribute := range object.Attributes {
					if !a.validContractType(name, contract, attribute.Type) {
						return validationError(object.Span, fmt.Sprintf("interface %s.%s type %s references unknown type %s", name, interfaceName, typeName, attribute.Type), "use a primitive, a type declared in this interface, or an exposed qualified type")
					}
				}
				if err := validateObjectOperations(typeName, object, func(typeName string) bool {
					return a.validContractType(name, contract, typeName)
				}); err != nil {
					return fmt.Errorf("interface %s.%s: %w", name, interfaceName, err)
				}
			}
			for operationName, operation := range contract.Operations {
				if operationName != operation.Name || operation.Name == "" {
					return fmt.Errorf("interface %s.%s has an invalid operation", name, interfaceName)
				}
				parameterNames := map[string]bool{}
				for _, parameter := range operation.Parameters {
					if parameter.Name == "" || parameterNames[parameter.Name] {
						return fmt.Errorf("interface %s.%s behavior %s has an invalid parameter", name, interfaceName, operationName)
					}
					parameterNames[parameter.Name] = true
					if !a.validContractType(name, contract, parameter.Type) {
						return fmt.Errorf("interface %s.%s behavior %s references unknown type %s", name, interfaceName, operationName, parameter.Type)
					}
				}
				if operation.Returns != "" && !a.validContractType(name, contract, operation.Returns) {
					return fmt.Errorf("interface %s.%s behavior %s references unknown return type %s", name, interfaceName, operationName, operation.Returns)
				}
			}
		}
		for exposed := range context.Exposes {
			if _, ok := context.Interfaces[exposed]; !ok {
				return validationError(context.ExposeSpans[exposed], fmt.Sprintf("context %s exposes undefined interface %s", name, exposed), suggestion("interface", exposed, context.Interfaces))
			}
		}
	}
	seenRelations := make(map[string]bool, len(a.Relations))
	for _, relation := range a.Relations {
		relationKey := relation.From + "\x00" + relation.To + "\x00" + relation.Via
		if seenRelations[relationKey] {
			return validationError(relation.Span, fmt.Sprintf("duplicate relationship from %s to %s", relation.From, relation.To), "declare a relationship only once")
		}
		seenRelations[relationKey] = true
		if _, ok := a.Contexts[relation.From]; !ok {
			return validationError(relation.Span, fmt.Sprintf("relationship references unknown context %s", relation.From), suggestion("context", relation.From, a.Contexts))
		}
		to, ok := a.Contexts[relation.To]
		if !ok {
			return validationError(relation.Span, fmt.Sprintf("relationship references unknown context %s", relation.To), suggestion("context", relation.To, a.Contexts))
		}
		if relation.From == relation.To {
			return fmt.Errorf("context %s cannot depend on itself", relation.From)
		}
		if relation.Via != "" {
			if !to.Exposes[relation.Via] {
				return fmt.Errorf("%s does not expose %s", relation.To, relation.Via)
			}
			if _, ok := to.Interfaces[relation.Via]; !ok {
				return fmt.Errorf("%s exposes undefined interface %s", relation.To, relation.Via)
			}
		}
	}
	if err := a.validateContextQuality(); err != nil {
		return err
	}
	return nil
}

func (a *Architecture) materializeModuleFiles() error {
	extension, ok := implementationExtension(a.Implementation.Language)
	if !ok {
		return validationError(a.ImplementationSpan, fmt.Sprintf("unsupported implementation language %s", a.Implementation.Language), "use one of the supported implementation languages: go, ruby, or python")
	}
	for index := range a.Files {
		if a.Files[index].RootEntrypoint {
			a.Files[index].Path = filepath.ToSlash(a.Files[index].EntryPointName + extension)
		}
	}
	mappedEntrypoints := make(map[string]bool)
	for _, file := range a.Files {
		if file.EntryPoint {
			mappedEntrypoints[file.Node+"."+file.EntryPointName] = true
		}
	}
	for _, module := range a.Modules {
		seen := map[string]bool{}
		for _, name := range module.Files {
			if seen[name] {
				return fmt.Errorf("module %s declares file %s more than once", module.Qualified, name)
			}
			seen[name] = true
			a.Files = append(a.Files, FileMapping{
				Path: filepath.ToSlash(filepath.Join(a.moduleFilePath(module), name+extension)),
				Node: module.Qualified,
			})
		}
		for entrypoint := range module.Entrypoints {
			key := module.Qualified + "." + entrypoint
			if mappedEntrypoints[key] {
				continue
			}
			a.Files = append(a.Files, FileMapping{
				Path:           filepath.ToSlash(filepath.Join(a.moduleEntrypointPath(module, entrypoint), "main"+extension)),
				Node:           module.Qualified,
				EntryPoint:     true,
				EntryPointName: entrypoint,
			})
			mappedEntrypoints[key] = true
		}
		module.Files = nil
	}
	return nil
}

func implementationExtension(language string) (string, bool) {
	extensions := map[string]string{
		"go":         ".go",
		"ruby":       ".rb",
		"python":     ".py",
		"rust":       ".rs",
		"typescript": ".ts",
	}
	extension, ok := extensions[language]
	return extension, ok
}

func (a *Architecture) modulePath(module *Module) string {
	parts := make([]string, 0)
	for current := module; current != nil; current = a.Modules[current.Parent] {
		parts = append(parts, ConventionalFolder(current.Name))
	}
	for left, right := 0, len(parts)-1; left < right; left, right = left+1, right-1 {
		parts[left], parts[right] = parts[right], parts[left]
	}
	return filepath.Join(parts...)
}

func (a *Architecture) moduleFilePath(module *Module) string {
	return a.modulePath(module)
}

func (a *Architecture) moduleEntrypointPath(module *Module, entrypoint string) string {
	if ConventionalFolder(module.Name) == "command" {
		return a.modulePath(module)
	}
	if ConventionalEntrypoint(module.Name) == ConventionalEntrypoint(entrypoint) {
		return a.modulePath(module)
	}
	return filepath.Join(a.modulePath(module), "command", ConventionalEntrypoint(entrypoint))
}

func (a *Architecture) validContractType(contextName string, contract *Interface, typeName string) bool {
	base := strings.TrimSuffix(typeName, "[]")
	primitives := map[string]bool{
		"any": true, "bool": true, "decimal": true, "float": true,
		"int": true, "string": true, "timestamp": true,
	}
	// Architecture-level domain types are reusable contract vocabulary. Keep
	// local contract types preferred so existing unqualified references retain
	// their meaning when names overlap.
	if primitives[base] || contract.Types[base] != nil || a.Objects[base] != nil {
		return true
	}
	parts := strings.Split(base, ".")
	if len(parts) == 2 {
		target := a.Contexts[contextName].Interfaces[parts[0]]
		return target != nil && target.Types[parts[1]] != nil
	}
	if len(parts) == 3 {
		context := a.Contexts[parts[0]]
		return context != nil &&
			context.Exposes[parts[1]] &&
			context.Interfaces[parts[1]].Types[parts[2]] != nil &&
			a.hasRelation(contextName, parts[0], parts[1])
	}
	return false
}

func (a *Architecture) validArchitectureType(typeName string) bool {
	base := strings.TrimSuffix(typeName, "[]")
	primitives := map[string]bool{
		"any": true, "bool": true, "decimal": true, "float": true,
		"int": true, "string": true, "timestamp": true,
	}
	return primitives[base] || a.Objects[base] != nil
}

func (a *Architecture) validModuleDependency(module *Module, dependency string) bool {
	if target := a.Modules[dependency]; target != nil {
		return target.Context == module.Context
	}
	if _, ok := a.Modules[module.Qualified+"."+dependency]; ok {
		return true
	}
	if module.Parent != "" {
		if _, ok := a.Modules[module.Parent+"."+dependency]; ok {
			return true
		}
	}
	if _, ok := a.Contexts[module.Context].Interfaces[dependency]; ok {
		return true
	}
	parts := strings.Split(dependency, ".")
	if len(parts) == 2 {
		if context := a.Contexts[parts[0]]; context != nil && context.Exposes[parts[1]] {
			return a.hasRelation(module.Context, parts[0], parts[1])
		}
	}
	return false
}

func (a *Architecture) validateContextQuality() error {
	contextNames := sortedContextNames(a.Contexts)
	if len(a.Contexts) > 1 {
		connected := make(map[string]bool, len(a.Contexts))
		for _, relation := range a.Relations {
			connected[relation.From] = true
			connected[relation.To] = true
		}
		for _, name := range contextNames {
			context := a.Contexts[name]
			if !connected[name] {
				return validationError(context.Span, fmt.Sprintf("context %s is unreachable", name), "connect it with a relationship or remove it")
			}
		}
	}

	for _, contextName := range contextNames {
		context := a.Contexts[contextName]
		for _, interfaceName := range sortedInterfaceNames(context.Interfaces) {
			used := context.Exposes[interfaceName]
			for _, contract := range context.Interfaces {
				for _, object := range contract.Types {
					for _, attribute := range object.Attributes {
						if strings.HasPrefix(strings.TrimSuffix(attribute.Type, "[]"), interfaceName+".") {
							used = true
						}
					}
					for _, operation := range object.Operations {
						if strings.HasPrefix(strings.TrimSuffix(operation.Returns, "[]"), interfaceName+".") {
							used = true
						}
						for _, parameter := range operation.Parameters {
							if strings.HasPrefix(strings.TrimSuffix(parameter.Type, "[]"), interfaceName+".") {
								used = true
							}
						}
					}
				}
				for _, operation := range contract.Operations {
					if strings.HasPrefix(strings.TrimSuffix(operation.Returns, "[]"), interfaceName+".") {
						used = true
					}
					for _, parameter := range operation.Parameters {
						if strings.HasPrefix(strings.TrimSuffix(parameter.Type, "[]"), interfaceName+".") {
							used = true
						}
					}
				}
			}
			for _, imported := range context.Imports {
				if imported.Symbol == interfaceName {
					used = true
					break
				}
			}
			for _, relation := range a.Relations {
				if relation.To == contextName && relation.Via == interfaceName {
					used = true
				}
			}
			for _, module := range a.Modules {
				if module.Context != contextName {
					continue
				}
				if module.Implements == interfaceName || module.Uses[interfaceName] || module.Uses[contextName+"."+interfaceName] {
					used = true
				}
			}
			if !used {
				return validationError(context.Interfaces[interfaceName].Span, fmt.Sprintf("interface %s.%s is unused", contextName, interfaceName), "expose it, implement it, or remove it")
			}
		}
	}

	adjacency := make(map[string][]string, len(a.Contexts))
	for _, relation := range a.Relations {
		adjacency[relation.From] = append(adjacency[relation.From], relation.To)
	}
	for name := range adjacency {
		sort.Strings(adjacency[name])
	}
	state := make(map[string]uint8, len(a.Contexts))
	var visit func(string) error
	visit = func(name string) error {
		state[name] = 1
		for _, target := range adjacency[name] {
			if state[target] == 1 {
				return fmt.Errorf("suspicious dependency cycle involving contexts %s and %s", name, target)
			}
			if state[target] == 0 {
				if err := visit(target); err != nil {
					return err
				}
			}
		}
		state[name] = 2
		return nil
	}
	for _, name := range contextNames {
		if state[name] == 0 {
			if err := visit(name); err != nil {
				return err
			}
		}
	}
	return nil
}

func sortedContextNames(contexts map[string]*Context) []string {
	names := make([]string, 0, len(contexts))
	for name := range contexts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedInterfaceNames(interfaces map[string]*Interface) []string {
	names := make([]string, 0, len(interfaces))
	for name := range interfaces {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (a *Architecture) hasRelation(from, to, via string) bool {
	for _, relation := range a.Relations {
		if relation.From == from && relation.To == to && relation.Via == via {
			return true
		}
	}
	return false
}

func validateObject(name string, object *Object) error {
	if object.Name == "" || object.Name != name || (object.Kind != "entity" && object.Kind != "value") {
		return fmt.Errorf("invalid domain type")
	}
	if object.Kind == "value" && len(object.Operations) > 0 {
		return fmt.Errorf("value %s cannot declare behavior", name)
	}
	for attributeName, attribute := range object.Attributes {
		if attribute.Name == "" || attribute.Name != attributeName || attribute.Type == "" {
			return fmt.Errorf("domain type %s has an invalid state", name)
		}
	}
	return nil
}

func validateObjectOperations(name string, object *Object, validType func(string) bool) error {
	for operationName, operation := range object.Operations {
		if operationName != operation.Name || operation.Name == "" {
			return fmt.Errorf("domain type %s has an invalid behavior", name)
		}
		parameterNames := map[string]bool{}
		for _, parameter := range operation.Parameters {
			if parameter.Name == "" || parameterNames[parameter.Name] {
				return fmt.Errorf("domain type %s behavior %s has an invalid parameter", name, operationName)
			}
			parameterNames[parameter.Name] = true
			if !validType(parameter.Type) {
				return fmt.Errorf("domain type %s behavior %s references unknown type %s", name, operationName, parameter.Type)
			}
		}
		if operation.Returns != "" && !validType(operation.Returns) {
			return fmt.Errorf("domain type %s behavior %s references unknown return type %s", name, operationName, operation.Returns)
		}
	}
	return nil
}

func (a *Architecture) HasNode(name string) bool {
	if _, ok := a.Modules[name]; ok {
		return true
	}
	if _, ok := a.Contexts[name]; ok {
		return true
	}
	for _, context := range a.Contexts {
		if _, ok := context.Interfaces[name]; ok {
			return true
		}
	}
	return false
}

func (a *Architecture) Allows(from, to string) bool {
	for _, relation := range a.Relations {
		if relation.From == from && relation.To == to {
			return true
		}
	}
	return false
}

func (a *Architecture) ModuleAllows(from, to string) bool {
	if from == to {
		return true
	}
	source, target := a.Modules[from], a.Modules[to]
	if source == nil || target == nil {
		return false
	}
	if source.Uses[to] {
		return source.Context == target.Context
	}
	for dependency := range source.Uses {
		if resolved := a.resolveModuleDependency(source, dependency); resolved == to {
			return true
		}
	}
	if target.Implements != "" && source.Uses[target.Implements] {
		if source.Context == target.Context {
			return true
		}
		for _, relation := range a.Relations {
			if relation.From == source.Context && relation.To == target.Context && relation.Via == target.Implements {
				return true
			}
		}
	}
	qualifiedInterface := target.Context + "." + target.Implements
	if target.Implements != "" && source.Uses[qualifiedInterface] {
		for _, relation := range a.Relations {
			if relation.From == source.Context && relation.To == target.Context && relation.Via == target.Implements {
				return true
			}
		}
	}
	return false
}

func (a *Architecture) resolveModuleDependency(source *Module, dependency string) string {
	if _, ok := a.Modules[dependency]; ok {
		return dependency
	}
	if candidate := source.Qualified + "." + dependency; a.Modules[candidate] != nil {
		return candidate
	}
	if source.Parent != "" {
		if candidate := source.Parent + "." + dependency; a.Modules[candidate] != nil {
			return candidate
		}
	}
	return ""
}
