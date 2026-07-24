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
	Imports        []string
}

type Module struct {
	Name        string
	Qualified   string
	Context     string
	Parent      string
	Implements  string
	Uses        map[string]bool
	Modules     map[string]*Module
	Entrypoints map[string]bool
}

// FileMapping assigns one implementation source file to one architecture context.
type FileMapping struct {
	Path           string
	Node           string
	EntryPoint     bool
	EntryPointName string
}

type Context struct {
	Name        string
	Description string
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
}

type Attribute struct {
	Name        string
	Type        string
	Description string
}
