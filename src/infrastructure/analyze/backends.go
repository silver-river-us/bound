package analyze

import "bound/src/lib/model"

// Backend validates an implementation tree against a Bound architecture.
// Language is the value used by the architecture's implementation declaration.
type Backend interface {
	Language() string
	Analyze(root string, architecture *model.Architecture) error
}

// GoBackend adapts the Go analyzer to the backend registry.
type GoBackend struct{}

func (GoBackend) Language() string { return "go" }
func (GoBackend) Analyze(root string, architecture *model.Architecture) error {
	return Go(root, architecture)
}

// RubyBackend adapts the Ruby analyzer to the backend registry.
type RubyBackend struct{}

func (RubyBackend) Language() string { return "ruby" }
func (RubyBackend) Analyze(root string, architecture *model.Architecture) error {
	return Ruby(root, architecture)
}

// PythonBackend adapts the Python analyzer to the backend registry.
type PythonBackend struct{}

func (PythonBackend) Language() string { return "python" }
func (PythonBackend) Analyze(root string, architecture *model.Architecture) error {
	return Python(root, architecture)
}
