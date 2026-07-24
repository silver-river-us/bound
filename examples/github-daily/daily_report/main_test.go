package dailyreport

import (
	"strings"
	"testing"
	"time"

	"github.com/silver-river-us/bound/examples/github-daily/github_activity"
)

func TestRenderReportGroupsOrganizations(t *testing.T) {
	when := time.Date(2026, 7, 24, 12, 0, 0, 0, time.UTC)
	report := RenderReport(when.Add(-24*time.Hour), when, []githubactivity.Organization{{Login: "acme"}, {Login: "empty"}}, []githubactivity.Activity{
		{Organization: "acme", Actor: "octocat", Repository: "acme/app", CreatedAt: when, Summary: "pushed 2 commit(s)"},
		{Organization: "acme", Actor: "GitHub", Repository: "acme/app", CreatedAt: when, Summary: "automated"},
		{Organization: "acme", Actor: "pull-request-actor[bot]", Repository: "acme/app", CreatedAt: when, Summary: "automated"},
	}, nil)
	for _, expected := range []string{"# GitHub activity report", "## acme (3 activities)", "octocat", "## empty (0 activities)", "No activity found", "2 automated/system activities"} {
		if !strings.Contains(report, expected) {
			t.Errorf("report does not contain %q", expected)
		}
	}
}
