package parser

import (
	"strings"
	"testing"
)

func TestArchitectureImplementation(t *testing.T) {
	architecture, err := Parse(strings.NewReader(`
architecture Example do
  implementation go "./"
  context Orders do
  end
end
`))
	if err != nil {
		t.Fatalf("parse architecture: %v", err)
	}

	if architecture.Implementation.Language != "go" {
		t.Fatalf("implementation language = %q, want go", architecture.Implementation.Language)
	}
	if architecture.Implementation.Locator != "./" {
		t.Fatalf("implementation locator = %q, want ./", architecture.Implementation.Locator)
	}
	if err := architecture.Validate(); err != nil {
		t.Fatalf("validate architecture: %v", err)
	}
}

func TestArchitectureImplementationIsRequiredFirst(t *testing.T) {
	_, err := Parse(strings.NewReader(`
architecture Example do
  context Orders do
  end
end
`))
	if err == nil || !strings.Contains(err.Error(), "expected architecture implementation") {
		t.Fatalf("error = %v, want expected architecture implementation", err)
	}
}

func TestNestedImplementationIsRejected(t *testing.T) {
	_, err := Parse(strings.NewReader(`
architecture Example do
  implementation go "./"
  context Orders do
    implementation go "./orders"
  end
end
`))
	if err == nil || !strings.Contains(err.Error(), "expected interface, module, exposes, or end") {
		t.Fatalf("error = %v, want nested implementation rejection", err)
	}
}

func TestDuplicateArchitectureImplementationIsRejected(t *testing.T) {
	_, err := Parse(strings.NewReader(`
architecture Example do
  implementation go "./"
  implementation go "./other"
end
`))
	if err == nil || !strings.Contains(err.Error(), "duplicate implementation") {
		t.Fatalf("error = %v, want duplicate implementation rejection", err)
	}
}
