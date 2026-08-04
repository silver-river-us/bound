package parser_test

import (
	. "bound/src/lib/parser"
	"errors"
	"fmt"
	"strings"
	"testing"

	"bound/src/lib/model"
)

func TestParserDiagnosticsIncludeSourceLocation(t *testing.T) {
	_, err := Parse(strings.NewReader("not architecture\n"))
	var parseError *Error
	if !errors.As(err, &parseError) || parseError.Line != 1 || parseError.Column != 1 {
		t.Fatalf("error = %#v, want parser location 1:1", err)
	}
}

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

func TestArchitectureQualityPolicy(t *testing.T) {
	architecture, err := Parse(strings.NewReader(`
architecture Example do
  implementation go "./"
  quality do
    max_function_lines 40
    max_cyclomatic_complexity 8
    max_nesting_depth 4
    max_parameters 5
    max_file_lines 300
    rules do
      one_declaration_kind_per_file
    end
  end
end
`))
	if err != nil {
		t.Fatalf("parse architecture: %v", err)
	}
	want := model.QualityPolicy{
		MaxFunctionLines:        40,
		MaxCyclomaticComplexity: 8,
		MaxNestingDepth:         4,
		MaxParameters:           5,
		MaxFileLines:            300,
		Rules:                   model.QualityRules{OneDeclarationKindPerFile: true},
	}
	if architecture.Quality != want {
		t.Fatalf("quality policy = %+v, want %+v", architecture.Quality, want)
	}
}

func TestArchitectureQualityPolicyRequiresPositiveKnownLimits(t *testing.T) {
	for _, rule := range []string{"max_function_lines 0", "unknown_quality_rule 4"} {
		t.Run(rule, func(t *testing.T) {
			_, err := Parse(strings.NewReader(fmt.Sprintf(`
architecture Example do
  implementation go "./"
  quality do
    %s
  end
end
`, rule)))
			if err == nil || !strings.Contains(err.Error(), "quality") {
				t.Fatalf("error = %v, want quality policy error", err)
			}
		})
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
	if err == nil || !strings.Contains(err.Error(), "expected import, interface, module, exposes, or end") {
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

func TestQualifiedImportNameIsRejected(t *testing.T) {
	_, err := Parse(strings.NewReader(`
architecture Example do
  implementation go "./"
  import Reporting.Activity from "activity.bo"
end
`))
	if err == nil || !strings.Contains(err.Error(), "expected context, relationship, or end") {
		t.Fatalf("error = %v, want qualified import rejection", err)
	}
}

func TestImportPathMustBeRelativeAndNonEmpty(t *testing.T) {
	for _, importPath := range []string{"", "../outside.bo", "contracts/../../outside.bo"} {
		t.Run(importPath, func(t *testing.T) {
			_, err := Parse(strings.NewReader(fmt.Sprintf(`
architecture Example do
  implementation go "./"
  import Reports from "%s"
end
`, importPath)))
			if err == nil || !strings.Contains(err.Error(), "import path") {
				t.Fatalf("error = %v, want invalid import path", err)
			}
		})
	}
}
