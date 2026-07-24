package model

type Architecture struct {
	Name        string
	Description string
	Contexts    map[string]*Context
	Objects     map[string]*Object
	Relations   []Relation
	Files       []FileMapping
	Imports     []string
}

// FileMapping assigns one implementation source file to one architecture context.
type FileMapping struct {
	Path       string
	Node       string
	EntryPoint bool
}

type Context struct {
	Name           string
	Description    string
	Implementation Implementation
	Exposes        map[string]bool
	Interfaces     map[string]*Interface
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
	Operations  map[string]Operation
}

type Operation struct {
	Name        string
	Signature   string
	Description string
}

type Object struct {
	Name        string
	Description string
	Attributes  map[string]Attribute
}

type Attribute struct {
	Name        string
	Type        string
	Description string
}
