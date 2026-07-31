package model

type Architecture struct {
	Name           string
	Description    string
	Implementation Implementation
	Contexts       map[string]*Context
	Objects        map[string]*Object
	Modules        map[string]*Module
	Relations      []Relation
	Files          []FileMapping
	Imports        []Import
	Quality        QualityPolicy
}

type QualityPolicy struct {
	MaxFunctionLines        int
	MaxCyclomaticComplexity int
	MaxNestingDepth         int
	MaxParameters           int
	MaxFileLines            int
	Rules                   QualityRules
}

type QualityRules struct {
	OneDeclarationKindPerFile bool
}

type Module struct {
	Name        string
	Qualified   string
	Context     string
	Parent      string
	Implements  string
	Uses        map[string]bool
	Files       []string
	Modules     map[string]*Module
	Entrypoints map[string]bool
}

type Import struct {
	Symbol string
	Kind   string
	Path   string
}

// FileMapping assigns one implementation source file to one architecture context.
type FileMapping struct {
	Path           string
	Node           string
	EntryPoint     bool
	EntryPointName string
	Explicit       bool
	RootEntrypoint bool
}

type Context struct {
	Name        string
	Description string
	Imports     []Import
	Exposes     map[string]bool
	Interfaces  map[string]*Interface
	Modules     map[string]*Module
}

type Implementation struct {
	Language string
	Locator  string
}

type Relation struct {
	From        string
	To          string
	Via         string
	Description string
}

type Interface struct {
	Name        string
	Description string
	Types       map[string]*Object
	Operations  map[string]Operation
}

type Operation struct {
	Name        string
	Signature   string
	Description string
	Parameters  []Parameter
	Returns     string
}

type Parameter struct {
	Name string
	Type string
}

type Object struct {
	Name        string
	Kind        string
	Description string
	Attributes  map[string]Attribute
	Operations  map[string]Operation
}

type Attribute struct {
	Name        string
	Type        string
	Description string
}
