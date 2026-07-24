package model

import "testing"

func TestConventionalFolderHandlesWordsAndAcronyms(t *testing.T) {
	tests := map[string]string{
		"DailyReport": "daily_report",
		"HTTPClient":  "http_client",
		"GitHub":      "git_hub",
		"Foo_Bar":     "foo_bar",
	}
	for input, want := range tests {
		if got := ConventionalFolder(input); got != want {
			t.Errorf("ConventionalFolder(%q) = %q, want %q", input, got, want)
		}
	}
}
