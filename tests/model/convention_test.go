package model_test

import (
	. "bound/src/lib/model"
	"testing"
)

func TestConventionalFolderHandlesWordsAndAcronyms(t *testing.T) {
	tests := map[string]string{
		"DailyReport":    "daily_report",
		"Command":        "command",
		"HTTPClient":     "http_client",
		"GitHub":         "github",
		"GitHubActivity": "github-activity",
		"Foo_Bar":        "foo_bar",
	}
	for input, want := range tests {
		if got := ConventionalFolder(input); got != want {
			t.Errorf("ConventionalFolder(%q) = %q, want %q", input, got, want)
		}
	}
}
