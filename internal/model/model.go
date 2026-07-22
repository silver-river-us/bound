package model

type Architecture struct {
	Name        string
	Description string
	Contexts    map[string]*Context
	Relations   []Relation
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
