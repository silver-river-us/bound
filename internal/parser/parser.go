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
	architectureRE   = regexp.MustCompile(`^architecture\s+([A-Za-z_][A-Za-z0-9_]*)\s*\{$`)
	contextRE        = regexp.MustCompile(`^context\s+([A-Za-z_][A-Za-z0-9_]*)\s*\{$`)
	implementationRE = regexp.MustCompile(`^implementation\s+([A-Za-z_][A-Za-z0-9_+-]*)\s+"([^"]+)"$`)
	exposesRE        = regexp.MustCompile(`^exposes\s+([A-Za-z_][A-Za-z0-9_]*)$`)
	relationRE       = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\s*->\s*([A-Za-z_][A-Za-z0-9_]*)(?:\s+via\s+([A-Za-z_][A-Za-z0-9_]*))?$`)
)

func Parse(r io.Reader) (*model.Architecture, error) {
	scanner := bufio.NewScanner(r)
	var architecture *model.Architecture
	var current *model.Context
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
		if current == nil && line == "}" {
			return architecture, nil
		}
		if current == nil {
			if match := contextRE.FindStringSubmatch(line); match != nil {
				if _, exists := architecture.Contexts[match[1]]; exists {
					return nil, syntaxError(lineNumber, "duplicate context")
				}
				current = &model.Context{Name: match[1], Exposes: map[string]bool{}}
				architecture.Contexts[current.Name] = current
				continue
			}
			if match := relationRE.FindStringSubmatch(line); match != nil {
				architecture.Relations = append(architecture.Relations, model.Relation{From: match[1], To: match[2], Via: match[3]})
				continue
			}
			return nil, syntaxError(lineNumber, "expected context, relationship, or closing brace")
		}
		if line == "}" {
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
		default:
			return nil, syntaxError(lineNumber, "expected implementation, exposes, or closing brace")
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if architecture == nil {
		return nil, fmt.Errorf("empty Bound document")
	}
	if current != nil {
		return nil, fmt.Errorf("line %d: unclosed context %s", lineNumber, current.Name)
	}
	return nil, fmt.Errorf("unclosed architecture %s", architecture.Name)
}

func syntaxError(line int, message string) error { return fmt.Errorf("line %d: %s", line, message) }
