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

	"bound/src/infrastructure/analyze"
	"bound/src/lib/model"
	"bound/src/lib/parser"
)

// Backend validates one implementation language.
type Backend interface {
	Language() string
	Analyze(root string, architecture *model.Architecture) error
}

// Registry resolves implementation languages to analyzers.
type Registry struct {
	backends map[string]Backend
}

// NewRegistry creates a backend registry. Later backends replace earlier
// backends with the same language, which makes explicit test or application
// overrides straightforward.
func NewRegistry(backends ...Backend) Registry {
	registry := Registry{backends: make(map[string]Backend, len(backends))}
	for _, backend := range backends {
		if backend != nil {
			registry.backends[backend.Language()] = backend
		}
	}
	return registry
}

// Register adds or replaces a backend in the registry.
func (r *Registry) Register(backend Backend) {
	if backend == nil {
		return
	}
	if r.backends == nil {
		r.backends = make(map[string]Backend)
	}
	r.backends[backend.Language()] = backend
}

// Lookup returns the analyzer registered for language.
func (r Registry) Lookup(language string) (Backend, bool) {
	backend, ok := r.backends[language]
	return backend, ok
}

// DefaultRegistry returns the built-in Go, Ruby, and Python backends.
func DefaultRegistry() Registry {
	return NewRegistry(analyze.GoBackend{}, analyze.RubyBackend{}, analyze.PythonBackend{})
}

type Options struct {
	// Backends overrides the built-in implementation backend registry.
	// A nil registry uses the built-in Go, Ruby, and Python backends.
	Backends *Registry
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
	Path    string `json:"path"`
	Line    int    `json:"line,omitempty"`
	Column  int    `json:"column,omitempty"`
	Message string `json:"message"`
}

type Diagnostic struct {
	Code       string              `json:"code"`
	Phase      string              `json:"phase"`
	Path       string              `json:"path"`
	Line       int                 `json:"line,omitempty"`
	Column     int                 `json:"column,omitempty"`
	Severity   string              `json:"severity"`
	Message    string              `json:"message"`
	Suggestion string              `json:"suggestion,omitempty"`
	Related    []RelatedDiagnostic `json:"related,omitempty"`
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
	message := fmt.Sprintf("%s %s%s: %s", d.Code, d.Phase, location, d.Message)
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
	registry := options.Backends
	if registry == nil {
		defaultBackends := DefaultRegistry()
		registry = &defaultBackends
	}
	if err := checkImplementation(program, *registry); err != nil {
		return nil, err
	}
	return program, nil
}

func checkImplementation(program *Program, registry Registry) error {
	language := program.Architecture.Implementation.Language
	backend, ok := registry.Lookup(language)
	if !ok {
		return diagnostic("analyze", program.SourceRoot, fmt.Errorf("no compiler backend registered for implementation language %q", language))
	}
	if err := backend.Analyze(program.SourceRoot, program.Architecture); err != nil {
		return diagnostic("analyze", program.SourceRoot, err)
	}
	return nil
}

func diagnostic(phase, path string, err error) error {
	diagnostic := Diagnostic{Code: diagnosticCode(phase), Phase: phase, Path: path, Severity: "error", Message: err.Error()}
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

func diagnosticCode(phase string) string {
	switch phase {
	case "resolve":
		return "BND100"
	case "parse":
		return "BND200"
	case "validate":
		return "BND300"
	case "analyze":
		return "BND400"
	default:
		return "BND000"
	}
}
