package render

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"bound/src/model"
)

// MermaidMarkdown renders the validated architecture as a Markdown document
// containing diagrams for context relationships, contracts and modules, the
// domain model, and source ownership.
func MermaidMarkdown(a *model.Architecture) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Architecture: %s\n\n", markdownText(a.Name))
	if a.Description != "" {
		fmt.Fprintf(&b, "%s\n\n", a.Description)
	}
	fmt.Fprintf(&b, "Implementation: `%s` at `%s`\n\n", markdownText(a.Implementation.Language), markdownText(a.Implementation.Locator))

	b.WriteString("## Context relationships\n\n```mermaid\n")
	b.WriteString(contextDiagram(a))
	b.WriteString("```\n\n## Component view\n\n```mermaid\n")
	b.WriteString(componentDiagram(a))
	b.WriteString("```\n\n## Interaction view (declared relationships)\n\n```mermaid\n")
	b.WriteString(interactionDiagram(a))
	b.WriteString("```\n\n## Contracts and modules\n\n```mermaid\n")
	b.WriteString(contractDiagram(a))
	b.WriteString("```\n\n## Domain model\n\n```mermaid\n")
	b.WriteString(domainDiagram(a))
	b.WriteString("```\n\n## Source ownership\n\n```mermaid\n")
	b.WriteString(sourceDiagram(a))
	b.WriteString("```\n")
	return b.String()
}

func componentDiagram(a *model.Architecture) string {
	var b strings.Builder
	b.WriteString("flowchart LR\n")
	for _, contextName := range sortedContextNames(a) {
		context := a.Contexts[contextName]
		fmt.Fprintf(&b, "  subgraph %s[\"%s\"]\n", mermaidID("component_context", contextName), mermaidText(contextName))
		fmt.Fprintf(&b, "    %s[\"%s\"]\n", mermaidID("component", contextName), mermaidText(contextName))
		for _, interfaceName := range sortedInterfaceNames(context) {
			fmt.Fprintf(&b, "    %s([\"%s\"])\n", mermaidID("port", contextName, interfaceName), mermaidText(interfaceName))
			fmt.Fprintf(&b, "    %s -. exposes .-> %s\n", mermaidID("component", contextName), mermaidID("port", contextName, interfaceName))
		}
		b.WriteString("  end\n")
	}
	for _, relation := range a.Relations {
		fmt.Fprintf(&b, "  %s -->|\"%s\"| %s\n", mermaidID("component", relation.From), mermaidText(relationLabel(relation)), mermaidID("component", relation.To))
	}
	return b.String()
}

// InteractionDiagram renders the architecture interaction sequence diagram.
func InteractionDiagram(a *model.Architecture) string {
	return interactionDiagram(a)
}

func interactionDiagram(a *model.Architecture) string {
	var b strings.Builder
	b.WriteString("sequenceDiagram\n")
	if len(a.Relations) > 0 {
		for _, contextName := range sortedContextNames(a) {
			fmt.Fprintf(&b, "  participant %s as %s\n", mermaidID("participant", contextName), mermaidText(contextName))
		}
		for _, relation := range a.Relations {
			fmt.Fprintf(&b, "  %s->>%s: %s\n", mermaidID("participant", relation.From), mermaidID("participant", relation.To), mermaidText(relationLabel(relation)))
		}
		return b.String()
	}

	participants := map[string]string{}
	type interaction struct {
		from  string
		to    string
		label string
	}
	interactions := make([]interaction, 0)
	for _, moduleName := range sortedModuleNamesAcrossArchitecture(a) {
		module := a.Modules[moduleName]
		if module == nil || len(module.Uses) == 0 {
			continue
		}
		context := a.Contexts[module.Context]
		for _, dependency := range sortedDependencies(module.Uses) {
			target := dependencyID(a, context, dependency)
			if target == "" {
				continue
			}
			participants[mermaidID("module", module.Qualified)] = module.Qualified
			participants[target] = dependencyLabel(a, context, dependency)
			interactions = append(interactions, interaction{from: mermaidID("module", module.Qualified), to: target, label: "uses " + dependencyLabel(a, context, dependency)})
		}
	}
	ids := make([]string, 0, len(participants))
	for id := range participants {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		fmt.Fprintf(&b, "  participant %s as %s\n", id, mermaidText(participants[id]))
	}
	if len(interactions) == 0 {
		b.WriteString("  participant Architecture\n  Note over Architecture: No declared interactions\n")
		return b.String()
	}
	for _, item := range interactions {
		fmt.Fprintf(&b, "  %s->>%s: %s\n", item.from, item.to, mermaidText(item.label))
	}
	return b.String()
}

func dependencyLabel(a *model.Architecture, context *model.Context, dependency string) string {
	if module := a.Modules[dependency]; module != nil {
		return module.Qualified
	}
	if _, exists := context.Interfaces[dependency]; exists {
		return dependency
	}
	prefix := context.Name + "."
	return strings.TrimPrefix(dependency, prefix)
}

func contextDiagram(a *model.Architecture) string {
	var b strings.Builder
	b.WriteString("classDiagram\n")
	for _, name := range sortedContextNames(a) {
		writeClass(&b, mermaidID("context", name), name, "context")
	}
	for _, relation := range a.Relations {
		label := relationLabel(relation)
		fmt.Fprintf(&b, "  %s --> %s : %s\n", mermaidID("context", relation.From), mermaidID("context", relation.To), mermaidText(label))
	}
	return b.String()
}

func contractDiagram(a *model.Architecture) string {
	var b strings.Builder
	b.WriteString("classDiagram\n")
	for _, contextName := range sortedContextNames(a) {
		context := a.Contexts[contextName]
		contextID := mermaidID("context", contextName)
		writeClass(&b, contextID, contextName, "context")
		for _, interfaceName := range sortedInterfaceNames(context) {
			interfaceID := mermaidID("interface", contextName, interfaceName)
			writeClass(&b, interfaceID, interfaceName, "interface")
			fmt.Fprintf(&b, "  %s o-- %s : owns\n", contextID, interfaceID)
			contract := context.Interfaces[interfaceName]
			for _, operationName := range sortedOperationNames(contract) {
				operation := contract.Operations[operationName]
				fmt.Fprintf(&b, "  %s : +%s%s\n", interfaceID, operation.Name, mermaidText(operation.Signature))
			}
		}
		for _, moduleName := range sortedModuleNames(a, context) {
			module := moduleForContext(a, context, moduleName)
			if module == nil {
				continue
			}
			writeClass(&b, mermaidID("module", module.Qualified), module.Qualified, "module")
			fmt.Fprintf(&b, "  %s o-- %s : owns\n", contextID, mermaidID("module", module.Qualified))
		}
	}
	for _, contextName := range sortedContextNames(a) {
		context := a.Contexts[contextName]
		for _, moduleName := range sortedModuleNames(a, context) {
			module := moduleForContext(a, context, moduleName)
			if module == nil {
				continue
			}
			if module.Implements != "" {
				fmt.Fprintf(&b, "  %s ..|> %s : implements\n", mermaidID("module", module.Qualified), mermaidID("interface", contextName, module.Implements))
			}
			dependencies := sortedDependencies(module.Uses)
			for _, dependency := range dependencies {
				if target := dependencyID(a, context, dependency); target != "" {
					fmt.Fprintf(&b, "  %s ..> %s : uses\n", mermaidID("module", module.Qualified), target)
				}
			}
		}
	}
	return b.String()
}

// DomainDiagram renders the architecture-level domain model diagram.
func DomainDiagram(a *model.Architecture) string {
	return domainDiagram(a)
}

func domainDiagram(a *model.Architecture) string {
	return domainDiagramObjects(a.Objects, nil)
}

// domainDiagramForContext renders the types owned by interfaces in one context.
// The architecture-level domain diagram intentionally remains unchanged; this
// scoped view is used by the HTML review to keep large models navigable.
func domainDiagramForContext(context *model.Context) string {
	objects := map[string]*model.Object{}
	for _, interfaceName := range sortedInterfaceNames(context) {
		contract := context.Interfaces[interfaceName]
		for _, typeName := range sortedObjectNamesForMap(contract.Types) {
			objects[interfaceName+"."+typeName] = contract.Types[typeName]
		}
	}
	return domainDiagramObjects(objects, func(name string) string { return name })
}

func domainDiagramObjects(objects map[string]*model.Object, labels func(string) string) string {
	var b strings.Builder
	b.WriteString("classDiagram\n")
	for _, name := range sortedObjectNamesForMap(objects) {
		object := objects[name]
		objectID := mermaidID("object", name)
		label := name
		if labels != nil {
			label = labels(name)
		}
		writeClass(&b, objectID, label, object.Kind)
		for _, attributeName := range sortedAttributeNames(object) {
			attribute := object.Attributes[attributeName]
			fmt.Fprintf(&b, "  %s : +%s %s\n", objectID, attribute.Name, mermaidText(attribute.Type))
			baseType := strings.TrimSuffix(attribute.Type, "[]")
			if target := domainObjectTarget(objects, name, baseType); target != "" {
				fmt.Fprintf(&b, "  %s --> %s : %s\n", objectID, mermaidID("object", target), mermaidText(attribute.Name))
			}
		}
		for _, operationName := range sortedOperationNamesForObject(object) {
			operation := object.Operations[operationName]
			fmt.Fprintf(&b, "  %s : +%s%s\n", objectID, operation.Name, mermaidText(operation.Signature))
		}
	}
	return b.String()
}

func domainObjectTarget(objects map[string]*model.Object, source, typeName string) string {
	if objects[typeName] != nil {
		return typeName
	}
	prefix := source
	if separator := strings.LastIndex(prefix, "."); separator >= 0 {
		prefix = prefix[:separator]
		candidate := prefix + "." + typeName
		if objects[candidate] != nil {
			return candidate
		}
	}
	var match string
	for name := range objects {
		if strings.HasSuffix(name, "."+typeName) {
			if match != "" {
				return ""
			}
			match = name
		}
	}
	return match
}

func sortedObjectNamesForMap(objects map[string]*model.Object) []string {
	names := make([]string, 0, len(objects))
	for name := range objects {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sourceDiagram(a *model.Architecture) string {
	var b strings.Builder
	b.WriteString("flowchart TB\n")
	files := append([]model.FileMapping(nil), a.Files...)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	filesByModule := make(map[string][]model.FileMapping)
	for _, file := range files {
		filesByModule[file.Node] = append(filesByModule[file.Node], file)
	}
	for _, name := range sortedModuleNamesAcrossArchitecture(a) {
		id, label := sourceOwner(a, name)
		fmt.Fprintf(&b, "  subgraph %s[\"%s\"]\n", id, mermaidText(label))
		b.WriteString("    direction TB\n")
		for _, file := range filesByModule[name] {
			label := filepath.Base(file.Path)
			if file.EntryPoint {
				label += " (entrypoint " + file.EntryPointName + ")"
			}
			fmt.Fprintf(&b, "    %s[\"%s\"]\n", mermaidID("file", file.Path), mermaidText(label))
		}
		b.WriteString("  end\n")
	}
	return b.String()
}

func sourceOwner(a *model.Architecture, name string) (string, string) {
	if name == "" {
		return mermaidID("source_module", "root"), a.Name + " (root)"
	}
	return mermaidID("source_module", name), name
}

func writeClass(b *strings.Builder, id, label, stereotype string) {
	fmt.Fprintf(b, "  class %s[\"%s\"]\n  <<%s>> %s\n", id, mermaidText(label), stereotype, id)
}

func sortedOperationNames(contract *model.Interface) []string {
	names := make([]string, 0, len(contract.Operations))
	for name := range contract.Operations {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedObjectNames(a *model.Architecture) []string {
	names := make([]string, 0, len(a.Objects))
	for name := range a.Objects {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedAttributeNames(object *model.Object) []string {
	names := make([]string, 0, len(object.Attributes))
	for name := range object.Attributes {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedOperationNamesForObject(object *model.Object) []string {
	names := make([]string, 0, len(object.Operations))
	for name := range object.Operations {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedContextNames(a *model.Architecture) []string {
	names := make([]string, 0, len(a.Contexts))
	for name := range a.Contexts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedInterfaceNames(context *model.Context) []string {
	names := make([]string, 0, len(context.Interfaces))
	for name := range context.Interfaces {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedModuleNames(a *model.Architecture, context *model.Context) []string {
	seen := map[string]bool{}
	for name := range context.Modules {
		seen[name] = true
	}
	for name, module := range a.Modules {
		if module.Context == context.Name {
			seen[name] = true
		}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func moduleForContext(a *model.Architecture, context *model.Context, name string) *model.Module {
	if module := context.Modules[name]; module != nil {
		return module
	}
	return a.Modules[name]
}

func sortedModuleNamesAcrossArchitecture(a *model.Architecture) []string {
	seen := make(map[string]bool, len(a.Modules))
	for name := range a.Modules {
		seen[name] = true
	}
	for _, file := range a.Files {
		seen[file.Node] = true
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func sortedDependencies(dependencies map[string]bool) []string {
	names := make([]string, 0, len(dependencies))
	for name := range dependencies {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func dependencyID(a *model.Architecture, context *model.Context, dependency string) string {
	if _, exists := a.Modules[dependency]; exists {
		return mermaidID("module", dependency)
	}
	for _, candidateContext := range a.Contexts {
		if _, exists := candidateContext.Modules[dependency]; exists {
			return mermaidID("module", dependency)
		}
	}
	if _, exists := context.Interfaces[dependency]; exists {
		return mermaidID("interface", context.Name, dependency)
	}
	prefix := context.Name + "."
	if strings.HasPrefix(dependency, prefix) {
		interfaceName := strings.TrimPrefix(dependency, prefix)
		if _, exists := context.Interfaces[interfaceName]; exists {
			return mermaidID("interface", context.Name, interfaceName)
		}
	}
	return ""
}

func relationLabel(relation model.Relation) string {
	if relation.Description != "" {
		return relation.Description
	}
	if relation.Via != "" {
		return "via " + relation.Via
	}
	return "depends on"
}

func mermaidID(prefix string, parts ...string) string {
	var b strings.Builder
	b.WriteString(prefix)
	for _, part := range parts {
		b.WriteByte('_')
		for _, character := range part {
			if (character >= 'a' && character <= 'z') || (character >= 'A' && character <= 'Z') || (character >= '0' && character <= '9') {
				b.WriteRune(character)
			} else {
				b.WriteByte('_')
			}
		}
	}
	return b.String()
}

func mermaidText(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "\"", "'"), "\n", " ")
}

func markdownText(value string) string {
	return strings.ReplaceAll(value, "\n", " ")
}
