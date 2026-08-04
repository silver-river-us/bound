package format_test

import (
	"strings"
	"testing"

	"bound/src/lib/format"
)

func TestFormatNormalizesStructuralIndentation(t *testing.T) {
	input := "architecture Example do\ncontext Boundary do\nmodule API do\nfile :api\nend\nend\nend\n"
	want := "architecture Example do\n  context Boundary do\n    module API do\n      file :api\n    end\n  end\nend\n"
	got, err := format.Format(input)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("formatted source = %q, want %q", got, want)
	}
}

func TestFormatPreservesDocumentationAndComments(t *testing.T) {
	input := "architecture Example do\n# keep this note\n\"\"\"\nA description.\n\"\"\"\nend\n"
	got, err := format.Format(input)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"# keep this note", "A description."} {
		if !strings.Contains(got, expected) {
			t.Fatalf("formatted source does not contain %q: %q", expected, got)
		}
	}
}

func TestFormatRejectsUnbalancedBlocks(t *testing.T) {
	if _, err := format.Format("architecture Example do\n"); err == nil {
		t.Fatal("expected unclosed block error")
	}
}
