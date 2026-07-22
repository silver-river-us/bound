package model

type Architecture struct {
	Name      string
	Contexts  map[string]*Context
	Relations []Relation
}

type Context struct {
	Name           string
	Implementation Implementation
	Exposes        map[string]bool
	Interfaces     map[string]*Interface
}

type Implementation struct {
	Language string
	Locator  string
}

type Relation struct {
	From string
	To   string
	Via  string
}

type Interface struct {
	Name       string
	Operations map[string]Operation
}

type Operation struct {
	Name      string
	Signature string
}
