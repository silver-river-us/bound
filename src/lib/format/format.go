// Package format provides the canonical whitespace formatter for Bound source.
package format

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Format returns canonical indentation for a Bound source document. Formatting
// is intentionally syntax-preserving: comments, blank lines, and descriptions
// are retained while structural blocks are normalized to two spaces.
func Format(source string) (string, error) {
	var output strings.Builder
	scanner := bufio.NewScanner(strings.NewReader(source))
	indent := 0
	document := false
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		raw := strings.TrimSpace(scanner.Text())
		if raw == "" {
			output.WriteByte('\n')
			continue
		}
		if raw == `"""` {
			writeIndented(&output, indent, raw)
			document = !document
			continue
		}
		if document {
			writeIndented(&output, indent, raw)
			continue
		}
		if raw == "end" {
			if indent == 0 {
				return "", fmt.Errorf("line %d: unexpected end", lineNumber)
			}
			indent--
		}
		writeIndented(&output, indent, raw)
		if opensBlock(raw) {
			indent++
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	if document {
		return "", fmt.Errorf("unterminated documentation block")
	}
	if indent != 0 {
		return "", fmt.Errorf("unclosed block")
	}
	return output.String(), nil
}

func writeIndented(output *strings.Builder, indent int, line string) {
	output.WriteString(strings.Repeat("  ", indent))
	output.WriteString(line)
	output.WriteByte('\n')
}

func opensBlock(line string) bool {
	if strings.HasSuffix(line, " do") {
		return true
	}
	return line == "quality" || line == "rules"
}

// FormatReader formats a source reader.
func FormatReader(reader io.Reader) (string, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return Format(string(content))
}
