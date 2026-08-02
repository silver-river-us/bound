// Package bound provides the public library API for compiling Bound
// architecture specifications.
package bound

import (
	"errors"
	"fmt"

	"bound/src/compiler"
)

// SchemaVersion is the version emitted by Program.JSON. Version 1 is stable:
// fields may be added, but existing fields will not change meaning within
// version 1. A breaking IR change increments this value.
const SchemaVersion = 1

// Options controls compilation behavior.
type Options struct {
	// SourceRoot overrides the implementation root declared by the architecture.
	// When empty, it is resolved relative to the architecture file.
	SourceRoot string
	// SkipImplementation validates the language-neutral model without running
	// the target-language implementation backend.
	SkipImplementation bool
}

// RelatedDiagnostic identifies another source location relevant to a diagnostic.
type RelatedDiagnostic struct {
	Path    string
	Line    int
	Column  int
	Message string
}

// Diagnostic describes one compiler failure. Phase is one of resolve, parse,
// validate, or analyze.
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

// Error reports one or more typed compiler diagnostics. Callers can use
// errors.As to inspect it without parsing the formatted error string.
type Error struct {
	diagnostics []Diagnostic
}

func (e *Error) Error() string {
	if len(e.diagnostics) == 0 {
		return "Bound compilation failed"
	}
	if len(e.diagnostics) == 1 {
		return e.diagnostics[0].Error()
	}
	message := "Bound compilation failed:"
	for _, diagnostic := range e.diagnostics {
		message += "\n- " + diagnostic.Error()
	}
	return message
}

// Diagnostics returns a copy of the compiler diagnostics.
func (e *Error) Diagnostics() []Diagnostic {
	return append([]Diagnostic(nil), e.diagnostics...)
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

// Program is the public, resolved result of a successful compilation. Its
// compiler representation is intentionally private so the library can evolve
// without exposing internal model packages.
type Program struct {
	path       string
	sourceRoot string
	compiler   *compiler.Program
}

// Path returns the absolute path of the compiled architecture file.
func (p *Program) Path() string { return p.path }

// SourceRoot returns the absolute implementation root used for analysis.
func (p *Program) SourceRoot() string { return p.sourceRoot }

// JSON returns the versioned compiler IR as a stable, human-readable document.
func (p *Program) JSON() ([]byte, error) {
	return p.compiler.JSON()
}

// Compile parses and validates an architecture specification and checks its
// declared implementation source tree. The path may be relative or absolute.
func Compile(path string, options Options) (*Program, error) {
	compiled, err := compiler.Compile(path, compiler.Options{
		SourceRoot:         options.SourceRoot,
		SkipImplementation: options.SkipImplementation,
	})
	if err != nil {
		var compilerError *compiler.Error
		if errors.As(err, &compilerError) {
			return nil, &Error{diagnostics: diagnosticsFromCompiler(compilerError)}
		}
		return nil, err
	}
	return &Program{
		path:       compiled.Path,
		sourceRoot: compiled.SourceRoot,
		compiler:   compiled,
	}, nil
}

func diagnosticsFromCompiler(source *compiler.Error) []Diagnostic {
	encoded := make([]Diagnostic, len(source.Diagnostics))
	for index, diagnostic := range source.Diagnostics {
		related := make([]RelatedDiagnostic, len(diagnostic.Related))
		for relatedIndex, source := range diagnostic.Related {
			related[relatedIndex] = RelatedDiagnostic{Path: source.Path, Line: source.Line, Column: source.Column, Message: source.Message}
		}
		encoded[index] = Diagnostic{
			Phase:      diagnostic.Phase,
			Path:       diagnostic.Path,
			Line:       diagnostic.Line,
			Column:     diagnostic.Column,
			Severity:   diagnostic.Severity,
			Message:    diagnostic.Message,
			Suggestion: diagnostic.Suggestion,
			Related:    related,
		}
	}
	return encoded
}
