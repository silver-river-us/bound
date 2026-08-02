// Package compiler coordinates the Bound front end and implementation
// backends. It is deliberately small for now: parsing and model validation
// produce the architecture IR, and language backends verify the implementation
// against that IR.
package compiler

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"

	"bound/src/analyze"
	"bound/src/model"
	"bound/src/parser"
)

type Options struct {
	// SourceRoot overrides the implementation root declared by the architecture.
	// When empty, it is resolved relative to the architecture file.
	SourceRoot string
	// SkipImplementation skips the target-language backend after model
	// validation. Compilation checks the implementation by default.
	SkipImplementation bool
}

type Program struct {
	Path         string              `json:"path"`
	SourceRoot   string              `json:"source_root"`
	Architecture *model.Architecture `json:"architecture"`
}

// JSON returns the resolved compiler IR as a stable, human-readable JSON
// document. The implementation is checked before the IR is returned unless
// Options.SkipImplementation was set during compilation.
func (p *Program) JSON() ([]byte, error) {
	return json.MarshalIndent(struct {
		SchemaVersion int      `json:"schema_version"`
		Program       *Program `json:"program"`
	}{SchemaVersion: 1, Program: p}, "", "  ")
}

type RelatedDiagnostic struct {
	Path    string
	Line    int
	Column  int
	Message string
}

type Diagnostic struct {
	Phase      string
	Path       string
	Line       int
	Column     int
	Severity   string
	Message    string
	Suggestion string
	Related    []RelatedDiagnostic
}

func (d Diagnostic) Error() string {
	location := ""
	if d.Path != "" {
		location = d.Path
		if d.Line > 0 {
			location += fmt.Sprintf(":%d:%d", d.Line, d.Column)
		}
		location += ": "
	}
	message := fmt.Sprintf("%s%s: %s", d.Phase, location, d.Message)
	if d.Suggestion != "" {
		message += " (suggestion: " + d.Suggestion + ")"
	}
	return message
}

type Error struct {
	Diagnostics []Diagnostic
}

func (e *Error) Error() string {
	if len(e.Diagnostics) == 0 {
		return "Bound compilation failed"
	}
	if len(e.Diagnostics) == 1 {
		return e.Diagnostics[0].Error()
	}
	message := "Bound compilation failed:"
	for _, diagnostic := range e.Diagnostics {
		message += "\n- " + diagnostic.Error()
	}
	return message
}

func (e *Error) Unwrap() error {
	if len(e.Diagnostics) == 1 {
		return errors.New(e.Diagnostics[0].Message)
	}
	return nil
}

// Compile parses and validates a Bound program, then checks its declared
// implementation dependencies with the backend for the target language.
func Compile(path string, options Options) (*Program, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return nil, diagnostic("resolve", path, err)
	}
	architecture, err := parser.ParseFile(absolutePath)
	if err != nil {
		return nil, diagnostic("parse", absolutePath, err)
	}
	if err := architecture.Validate(); err != nil {
		return nil, diagnostic("validate", absolutePath, err)
	}

	root := options.SourceRoot
	if root == "" {
		root = filepath.Join(filepath.Dir(absolutePath), architecture.Implementation.Locator)
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return nil, diagnostic("resolve", absolutePath, fmt.Errorf("source root: %w", err))
	}
	program := &Program{Path: absolutePath, SourceRoot: root, Architecture: architecture}
	if options.SkipImplementation {
		return program, nil
	}
	if err := checkImplementation(program); err != nil {
		return nil, err
	}
	return program, nil
}

func checkImplementation(program *Program) error {
	switch program.Architecture.Implementation.Language {
	case "go":
		if err := analyze.Go(program.SourceRoot, program.Architecture); err != nil {
			return diagnostic("analyze", program.SourceRoot, err)
		}
	case "ruby":
		if err := analyze.Ruby(program.SourceRoot, program.Architecture); err != nil {
			return diagnostic("analyze", program.SourceRoot, err)
		}
	case "python":
		if err := analyze.Python(program.SourceRoot, program.Architecture); err != nil {
			return diagnostic("analyze", program.SourceRoot, err)
		}
	default:
		return diagnostic("analyze", program.SourceRoot, fmt.Errorf("no compiler backend for implementation language %q", program.Architecture.Implementation.Language))
	}
	return nil
}

func diagnostic(phase, path string, err error) error {
	diagnostic := Diagnostic{Phase: phase, Path: path, Severity: "error", Message: err.Error()}
	var parserError *parser.Error
	if errors.As(err, &parserError) {
		diagnostic.Line, diagnostic.Column = parserError.Line, parserError.Column
		diagnostic.Message = parserError.Message
	}
	var modelError *model.Error
	if errors.As(err, &modelError) {
		diagnostic.Line, diagnostic.Column = modelError.Span.Line, modelError.Span.Column
		diagnostic.Message = modelError.Message
		diagnostic.Suggestion = modelError.Suggestion
	}
	return &Error{Diagnostics: []Diagnostic{diagnostic}}
}
