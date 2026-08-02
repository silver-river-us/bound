// Package cli is the command-line boundary for the GitHub daily application.
package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	githubapi "bound/examples/github-activity/infrastructure/github"
	"bound/examples/github-activity/lib/reporting"
	"bound/src/framework"
)

func Run() {
	options := parseFlags()
	validateArchitecture(options)
	result := generateReport(options)
	if result.Err != nil {
		fatal(result.Err)
	}
	writeReport(options.output, RenderMarkdown(result.Report), result.ActivityCount, result.OrganizationCount)
}

func validateArchitecture(options options) {
	if err := framework.CheckArchitecture(options.architecture, options.sourceRoot); err != nil {
		fatal(err)
	}
}

func generateReport(options options) reporting.Result {
	token, err := githubapi.AuthToken()
	if err != nil {
		return reporting.Result{Err: err}
	}
	client := githubapi.NewClient(options.baseURL, token)
	return reporting.Generate(client, time.Now().UTC().Add(-options.period), time.Now().UTC())
}

func parseFlags() options {
	defaultOutput := filepath.Join("examples", "github-activity", "reports", "github-activity-"+time.Now().UTC().Format("2006-01-02")+".md")
	period := flag.Duration("period", 24*time.Hour, "activity period, for example 24h or 48h")
	output := flag.String("output", defaultOutput, "Markdown report path; use - for stdout")
	baseURL := flag.String("base-url", githubapi.GithubAPI, "GitHub API base URL")
	architecture := flag.String("architecture", filepath.Join("examples", "github-activity", "github-activity.bo"), "Bound architecture file")
	sourceRoot := flag.String("source-root", filepath.Join("examples", "github-activity"), "source root checked by Bound")
	flag.Parse()
	return options{period: *period, output: *output, baseURL: *baseURL, architecture: *architecture, sourceRoot: *sourceRoot}
}

func writeReport(output, report string, activityCount, organizationCount int) {
	if output == "-" {
		fmt.Print(report)
		return
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		fatal(err)
	}
	if err := os.WriteFile(output, []byte(report), 0o644); err != nil {
		fatal(err)
	}
	logger.Printf("wrote %d activities across %d organizations to %s", activityCount, organizationCount, output)
}
