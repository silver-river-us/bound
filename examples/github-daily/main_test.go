package main

import (
	"strings"
	"testing"
	"time"
)

func TestRenderReportGroupsOrganizations(t *testing.T) {
	when := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	report := renderReport(when.Add(-24*time.Hour), when, []organization{{Login: "acme"}, {Login: "empty"}}, []activity{{Organization: "acme", Actor: "octocat", Repository: "acme/app", CreatedAt: when, Summary: "pushed 2 commit(s)"}}, nil)
	for _, expected := range []string{"# GitHub activity report", "## acme (1 activities)", "octocat", "## empty (0 activities)", "No activity found"} {
		if !strings.Contains(report, expected) {
			t.Errorf("report does not contain %q", expected)
		}
	}
}
