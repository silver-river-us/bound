package parser

import (
	"bufio"
	"fmt"
	"github.com/silver-river-us/bound/internal/model"
	"io"
	"regexp"
	"strings"
)

var (
	architectureRE   = regexp.MustCompile(`^architecture\s+([A-Za-z_][A-Za-z0-9_]*)\s+do$`)
	objectRE         = regexp.MustCompile(`^object\s+([A-Za-z_][A-Za-z0-9_]*)\s+do$`)
	attributeRE      = regexp.MustCompile(`^attribute\s+:([A-Za-z_][A-Za-z0-9_]*)\s+:([A-Za-z_][A-Za-z0-9_\[\]]*)$`)
	filesRE          = regexp.MustCompile(`^files\s+do$`)
	fileMappingRE    = regexp.MustCompile(`^"([^"]+)"\s*->\s*([A-Za-z_][A-Za-z0-9_]*)$`)
	entrypointRE     = regexp.MustCompile(`^entrypoint\s+"([^"]+)"\s*->\s*([A-Za-z_][A-Za-z0-9_]*)$`)
	contextRE        = regexp.MustCompile(`^context\s+([A-Za-z_][A-Za-z0-9_]*)\s+do$`)
	implementationRE = regexp.MustCompile(`^implementation\s+([A-Za-z_][A-Za-z0-9_+-]*)\s+"([^"]+)"$`)
	exposesRE        = regexp.MustCompile(`^exposes\s+([A-Za-z_][A-Za-z0-9_]*)$`)
	interfaceRE      = regexp.MustCompile(`^interface\s+([A-Za-z_][A-Za-z0-9_]*)\s+do$`)
	operationRE      = regexp.MustCompile(`^operation\s+([A-Za-z_][A-Za-z0-9_]*)(.*)$`)
	relationRE       = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*->\s*([A-Za-z_][A-Za-z0-9_]*)(?:\s+via\s+([A-Za-z_][A-Za-z0-9_]*))?$`)
)

func Parse(r io.Reader) (*model.Architecture, error) {
	scanner := bufio.NewScanner(r)
	var architecture *model.Architecture
	var current *model.Context
	var currentInterface *model.Interface
	var currentObject *model.Object
	currentFiles := false
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
		if architecture == nil {
			match := architectureRE.FindStringSubmatch(line)
			if match == nil {
				return nil, syntaxError(lineNumber, "expected architecture declaration")
			}
			architecture = &model.Architecture{Name: match[1], Description: pendingDescription, Contexts: map[string]*model.Context{}, Objects: map[string]*model.Object{}}
			pendingDescription = ""
			continue
		}
		if current == nil && currentObject != nil {
			if line == "end" {
				currentObject = nil
				continue
			}
			match := attributeRE.FindStringSubmatch(line)
			if match == nil {
				return nil, syntaxError(lineNumber, "expected attribute or end")
			}
			if _, exists := currentObject.Attributes[match[1]]; exists {
				return nil, syntaxError(lineNumber, "duplicate attribute")
			}
			currentObject.Attributes[match[1]] = model.Attribute{Name: match[1], Type: match[2], Description: pendingDescription}
			pendingDescription = ""
			continue
		}
		if current == nil && currentObject == nil && currentFiles {
			if line == "end" {
				currentFiles = false
				continue
			}
			if match := fileMappingRE.FindStringSubmatch(line); match != nil {
				architecture.Files = append(architecture.Files, model.FileMapping{Path: match[1], Node: match[2]})
				continue
			}
			if match := entrypointRE.FindStringSubmatch(line); match != nil {
				architecture.Files = append(architecture.Files, model.FileMapping{Path: match[1], Node: match[2], EntryPoint: true})
				continue
			}
			return nil, syntaxError(lineNumber, "expected file mapping, entrypoint, or end")
		}
		if current == nil && line == "end" {
			return architecture, nil
		}
		if current == nil {
			if filesRE.MatchString(line) {
				currentFiles = true
				continue
			}
			if match := objectRE.FindStringSubmatch(line); match != nil {
				if _, exists := architecture.Objects[match[1]]; exists {
					return nil, syntaxError(lineNumber, "duplicate object")
				}
				currentObject = &model.Object{Name: match[1], Description: pendingDescription, Attributes: map[string]model.Attribute{}}
				pendingDescription = ""
				architecture.Objects[match[1]] = currentObject
				continue
			}
			if match := contextRE.FindStringSubmatch(line); match != nil {
				if _, exists := architecture.Contexts[match[1]]; exists {
					return nil, syntaxError(lineNumber, "duplicate context")
				}
				current = &model.Context{Name: match[1], Description: pendingDescription, Exposes: map[string]bool{}, Interfaces: map[string]*model.Interface{}}
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
			match := operationRE.FindStringSubmatch(line)
			if match == nil {
				return nil, syntaxError(lineNumber, "expected operation or end")
			}
			if _, exists := currentInterface.Operations[match[1]]; exists {
				return nil, syntaxError(lineNumber, "duplicate operation")
			}
			currentInterface.Operations[match[1]] = model.Operation{Name: match[1], Signature: strings.TrimSpace(match[2]), Description: pendingDescription}
			pendingDescription = ""
			continue
		}
		if line == "end" {
			current = nil
			continue
		}
		switch {
		case implementationRE.MatchString(line):
			match := implementationRE.FindStringSubmatch(line)
			current.Implementation.Language = match[1]
			current.Implementation.Locator = match[2]
		case exposesRE.MatchString(line):
			current.Exposes[exposesRE.FindStringSubmatch(line)[1]] = true
		case interfaceRE.MatchString(line):
			match := interfaceRE.FindStringSubmatch(line)
			if _, exists := current.Interfaces[match[1]]; exists {
				return nil, syntaxError(lineNumber, "duplicate interface")
			}
			currentInterface = &model.Interface{Name: match[1], Description: pendingDescription, Operations: map[string]model.Operation{}}
			pendingDescription = ""
			current.Interfaces[match[1]] = currentInterface
		default:
			return nil, syntaxError(lineNumber, "expected implementation, exposes, or end")
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if architecture == nil {
		return nil, fmt.Errorf("empty Bound document")
	}
	if currentObject != nil {
		return nil, fmt.Errorf("line %d: unclosed object %s", lineNumber, currentObject.Name)
	}
	if current != nil {
		if currentInterface != nil {
			return nil, fmt.Errorf("line %d: unclosed interface %s", lineNumber, currentInterface.Name)
		}
		return nil, fmt.Errorf("line %d: unclosed context %s", lineNumber, current.Name)
	}
	if currentFiles {
		return nil, fmt.Errorf("line %d: unclosed files block", lineNumber)
	}
	return nil, fmt.Errorf("unclosed architecture %s", architecture.Name)
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
