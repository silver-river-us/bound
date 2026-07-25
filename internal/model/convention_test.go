package model

import "testing"

func TestConventionalFolderHandlesWordsAndAcronyms(t *testing.T) {
	tests := map[string]string{
		"DailyReport": "daily_report",
		"Command":     "command",
		"HTTPClient":  "http_client",
		"GitHub":      "github",
		"GitHubDaily": "github-daily",
		"Foo_Bar":     "foo_bar",
	}
	for input, want := range tests {
		if got := ConventionalFolder(input); got != want {
			t.Errorf("ConventionalFolder(%q) = %q, want %q", input, got, want)
		}
	}
}
