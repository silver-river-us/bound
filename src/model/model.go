package model

type Span struct {
	Line   int `json:"line,omitempty"`
	Column int `json:"column,omitempty"`
}

type Error struct {
	Message    string
	Span       Span
	Suggestion string
}

func (e *Error) Error() string { return e.Message }

type Architecture struct {
	Name               string              `json:"name"`
	Description        string              `json:"description,omitempty"`
	Implementation     Implementation      `json:"implementation"`
	ImplementationSpan Span                `json:"-"`
	Contexts           map[string]*Context `json:"contexts"`
	Objects            map[string]*Object  `json:"objects"`
	Modules            map[string]*Module  `json:"modules"`
	Relations          []Relation          `json:"relations,omitempty"`
	Files              []FileMapping       `json:"files,omitempty"`
	Imports            []Import            `json:"imports,omitempty"`
	Quality            QualityPolicy       `json:"quality"`
	Span               Span                `json:"-"`
}

type QualityPolicy struct {
	MaxFunctionLines        int          `json:"max_function_lines,omitempty"`
	MaxCyclomaticComplexity int          `json:"max_cyclomatic_complexity,omitempty"`
	MaxNestingDepth         int          `json:"max_nesting_depth,omitempty"`
	MaxParameters           int          `json:"max_parameters,omitempty"`
	MaxFileLines            int          `json:"max_file_lines,omitempty"`
	Rules                   QualityRules `json:"rules"`
}

type QualityRules struct {
	OneDeclarationKindPerFile bool `json:"one_declaration_kind_per_file,omitempty"`
}

type Module struct {
	Name        string             `json:"name"`
	Qualified   string             `json:"qualified"`
	Context     string             `json:"context"`
	Parent      string             `json:"parent,omitempty"`
	Implements  string             `json:"implements,omitempty"`
	Uses        map[string]bool    `json:"uses,omitempty"`
	Files       []string           `json:"files,omitempty"`
	Modules     map[string]*Module `json:"modules,omitempty"`
	Entrypoints map[string]bool    `json:"entrypoints,omitempty"`
	Span        Span               `json:"-"`
}

type Import struct {
	Symbol string `json:"symbol"`
	Kind   string `json:"kind"`
	Path   string `json:"path"`
}

// FileMapping assigns one implementation source file to one architecture context.
type FileMapping struct {
	Path           string `json:"path"`
	Node           string `json:"node,omitempty"`
	EntryPoint     bool   `json:"entrypoint,omitempty"`
	EntryPointName string `json:"entrypoint_name,omitempty"`
	Explicit       bool   `json:"explicit,omitempty"`
	RootEntrypoint bool   `json:"root_entrypoint,omitempty"`
}

type Context struct {
	Name        string                `json:"name"`
	Description string                `json:"description,omitempty"`
	Imports     []Import              `json:"imports,omitempty"`
	Exposes     map[string]bool       `json:"exposes,omitempty"`
	ExposeSpans map[string]Span       `json:"-"`
	Interfaces  map[string]*Interface `json:"interfaces,omitempty"`
	Modules     map[string]*Module    `json:"modules,omitempty"`
	Span        Span                  `json:"-"`
}

type Implementation struct {
	Language string `json:"language"`
	Locator  string `json:"locator"`
}

type Relation struct {
	From        string `json:"from"`
	To          string `json:"to"`
	Via         string `json:"via,omitempty"`
	Description string `json:"description,omitempty"`
	Span        Span   `json:"-"`
}

type Interface struct {
	Name        string               `json:"name"`
	Description string               `json:"description,omitempty"`
	Types       map[string]*Object   `json:"types,omitempty"`
	Operations  map[string]Operation `json:"operations,omitempty"`
	Span        Span                 `json:"-"`
}

type Operation struct {
	Name        string      `json:"name"`
	Signature   string      `json:"signature"`
	Description string      `json:"description,omitempty"`
	Parameters  []Parameter `json:"parameters,omitempty"`
	Returns     string      `json:"returns,omitempty"`
}

type Parameter struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type Object struct {
	Name        string               `json:"name"`
	Kind        string               `json:"kind"`
	Description string               `json:"description,omitempty"`
	Attributes  map[string]Attribute `json:"attributes,omitempty"`
	Operations  map[string]Operation `json:"operations,omitempty"`
	Span        Span                 `json:"-"`
}

type Attribute struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}
