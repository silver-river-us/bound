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
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(strings.SplitN(scanner.Text(), "#", 2)[0])
		if line == "" {
			continue
		}
		if architecture == nil {
			match := architectureRE.FindStringSubmatch(line)
			if match == nil {
				return nil, syntaxError(lineNumber, "expected architecture declaration")
			}
			architecture = &model.Architecture{Name: match[1], Contexts: map[string]*model.Context{}}
			continue
		}
		if current == nil && line == "end" {
			return architecture, nil
		}
		if current == nil {
			if match := contextRE.FindStringSubmatch(line); match != nil {
				if _, exists := architecture.Contexts[match[1]]; exists {
					return nil, syntaxError(lineNumber, "duplicate context")
				}
				current = &model.Context{Name: match[1], Exposes: map[string]bool{}, Interfaces: map[string]*model.Interface{}}
				architecture.Contexts[current.Name] = current
				continue
			}
			if match := relationRE.FindStringSubmatch(line); match != nil {
				architecture.Relations = append(architecture.Relations, model.Relation{From: match[1], To: match[2], Via: match[3]})
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
			currentInterface.Operations[match[1]] = model.Operation{Name: match[1], Signature: strings.TrimSpace(match[2])}
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
			currentInterface = &model.Interface{Name: match[1], Operations: map[string]model.Operation{}}
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
	if current != nil {
		if currentInterface != nil {
			return nil, fmt.Errorf("line %d: unclosed interface %s", lineNumber, currentInterface.Name)
		}
		return nil, fmt.Errorf("line %d: unclosed context %s", lineNumber, current.Name)
	}
	return nil, fmt.Errorf("unclosed architecture %s", architecture.Name)
}

func syntaxError(line int, message string) error { return fmt.Errorf("line %d: %s", line, message) }
