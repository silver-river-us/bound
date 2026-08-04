package analyze_test

import (
	. "bound/src/infrastructure/analyze"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bound/src/lib/model"
)

func TestValidateGoQualityRejectsConfiguredMetrics(t *testing.T) {
	tests := []struct {
		name   string
		source string
		policy model.QualityPolicy
		want   string
	}{
		{
			name: "function length",
			source: `package example

func tooLong() {
		one := 1
		two := 2
		three := 3
		_ = one + two + three
}
`,
			policy: model.QualityPolicy{MaxFunctionLines: 3},
			want:   "function tooLong is 6 lines, limit 3",
		},
		{
			name: "complexity",
			source: `package example

func tooComplex(value int) bool {
		if value > 0 && value < 10 {
			return true
		}
		return false
}
`,
			policy: model.QualityPolicy{MaxCyclomaticComplexity: 2},
			want:   "function tooComplex has cyclomatic complexity 3, limit 2",
		},
		{
			name: "nesting",
			source: `package example

func tooNested(value int) int {
		if value > 0 {
			for i := 0; i < value; i++ {
				if i > 1 {
					return i
				}
			}
		}
		return 0
}
`,
			policy: model.QualityPolicy{MaxNestingDepth: 2},
			want:   "function tooNested has nesting depth 3, limit 2",
		},
		{
			name: "parameters",
			source: `package example

func tooMany(a, b, c int) {}
`,
			policy: model.QualityPolicy{MaxParameters: 2},
			want:   "function tooMany has 3 parameters, limit 2",
		},
		{
			name: "file length",
			source: `package example

func one() {}
func two() {}
func three() {}
`,
			policy: model.QualityPolicy{MaxFileLines: 4},
			want:   "file is 5 lines, limit 4",
		},
		{
			name: "declaration kinds",
			source: `package example

type Record struct{}

func newRecord() Record { return Record{} }
`,
			policy: model.QualityPolicy{Rules: model.QualityRules{OneDeclarationKindPerFile: true}},
			want:   "file mixes top-level declaration kinds (functions, types)",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			if err := os.WriteFile(filepath.Join(directory, "main.go"), []byte(test.source), 0o600); err != nil {
				t.Fatalf("write source: %v", err)
			}
			err := ValidateGoQuality(directory, &model.Architecture{Quality: test.policy}, map[string]model.FileMapping{
				"main.go": {Path: "main.go"},
			})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want %q", err, test.want)
			}
		})
	}
}
