// Command github-daily writes a Markdown activity report for every GitHub
// organization visible to the authenticated user.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/silver-river-us/bound/examples/github-daily/daily_reporting/daily_report"
	"github.com/silver-river-us/bound/examples/github-daily/daily_reporting/github_activity"
	githubapi "github.com/silver-river-us/bound/examples/github-daily/daily_reporting/github_activity/github"
)

func main() {
	defaultOutput := filepath.Join("reports", "github-activity-"+time.Now().UTC().Format("2006-01-02")+".md")
	sinceFlag := flag.Duration("since", 24*time.Hour, "report window, for example 24h or 48h")
	outputFlag := flag.String("output", defaultOutput, "Markdown report path; use - for stdout")
	baseURLFlag := flag.String("base-url", githubapi.GithubAPI, "GitHub API base URL")
	architectureFlag := flag.String("architecture", filepath.Join("examples", "github-daily", "github-daily.bo"), "Bound architecture file")
	sourceRootFlag := flag.String("source-root", "examples/github-daily", "source root checked by Bound")
	flag.Parse()
	if err := checkArchitecture(*architectureFlag, *sourceRootFlag); err != nil {
		fatal(err)
	}

	token, err := githubapi.AuthToken()
	if err != nil {
		fatal(err)
	}
	c := githubapi.NewClient(*baseURLFlag, token)
	since := time.Now().UTC().Add(-*sinceFlag)
	orgs, err := c.Organizations()
	if err != nil {
		fatal(err)
	}

	activities := make([]githubactivity.Activity, 0)
	warnings := make([]string, 0)
	for _, org := range orgs {
		items, orgWarnings := c.Activities(org.Login, since, time.Now().UTC())
		activities = append(activities, items...)
		warnings = append(warnings, orgWarnings...)
	}
	sort.Slice(activities, func(i, j int) bool { return activities[i].CreatedAt.Before(activities[j].CreatedAt) })

	report := dailyreport.RenderReport(since, time.Now().UTC(), orgs, activities, warnings)
	if *outputFlag == "-" {
		fmt.Print(report)
		return
	}
	if err := os.MkdirAll(filepath.Dir(*outputFlag), 0o755); err != nil {
		fatal(err)
	}
	if err := os.WriteFile(*outputFlag, []byte(report), 0o644); err != nil {
		fatal(err)
	}
	fmt.Printf("wrote %d activities across %d organizations to %s\n", len(activities), len(orgs), *outputFlag)
}
