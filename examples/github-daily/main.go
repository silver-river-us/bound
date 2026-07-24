// Command github-daily writes a Markdown activity report for every GitHub
// organization visible to the authenticated user.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/silver-river-us/bound/internal/analyze"
	"github.com/silver-river-us/bound/internal/parser"
)

const githubAPI = "https://api.github.com"

type client struct {
	baseURL string
	token   string
	http    *http.Client
}

type organization struct {
	Login string `json:"login"`
}

type event struct {
	Type  string `json:"type"`
	Actor struct {
		Login string `json:"login"`
	} `json:"actor"`
	Repo struct {
		Name string `json:"name"`
	} `json:"repo"`
	CreatedAt time.Time       `json:"created_at"`
	Payload   json.RawMessage `json:"payload"`
}

type activity struct {
	Organization string
	Source       string
	Type         string
	Actor        string
	Repository   string
	CreatedAt    time.Time
	Summary      string
	URL          string
}

type searchResponse[T any] struct {
	TotalCount int `json:"total_count"`
	Items      []T `json:"items"`
}

type issueResult struct {
	Title         string    `json:"title"`
	HTMLURL       string    `json:"html_url"`
	UpdatedAt     time.Time `json:"updated_at"`
	RepositoryURL string    `json:"repository_url"`
	User          struct {
		Login string `json:"login"`
	} `json:"user"`
	PullRequest *struct{} `json:"pull_request"`
}

type commitResult struct {
	HTMLURL string `json:"html_url"`
	Author  struct {
		Login string `json:"login"`
	} `json:"author"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	Commit struct {
		Message string `json:"message"`
		Author  struct {
			Name string    `json:"name"`
			Date time.Time `json:"date"`
		} `json:"author"`
		Committer struct {
			Name string    `json:"name"`
			Date time.Time `json:"date"`
		} `json:"committer"`
	} `json:"commit"`
}

func main() {
	defaultOutput := filepath.Join("reports", "github-activity-"+time.Now().UTC().Format("2006-01-02")+".md")
	sinceFlag := flag.Duration("since", 24*time.Hour, "report window, for example 24h or 48h")
	outputFlag := flag.String("output", defaultOutput, "Markdown report path; use - for stdout")
	baseURLFlag := flag.String("base-url", githubAPI, "GitHub API base URL")
	architectureFlag := flag.String("architecture", filepath.Join("examples", "github-daily", "github-daily.bo"), "Bound architecture file")
	sourceRootFlag := flag.String("source-root", "examples/github-daily", "source root checked by Bound")
	flag.Parse()
	if err := checkArchitecture(*architectureFlag, *sourceRootFlag); err != nil {
		fatal(err)
	}

	token, err := authToken()
	if err != nil {
		fatal(err)
	}
	c := &client{baseURL: strings.TrimRight(*baseURLFlag, "/"), token: token, http: http.DefaultClient}
	since := time.Now().UTC().Add(-*sinceFlag)
	orgs, err := c.organizations()
	if err != nil {
		fatal(err)
	}

	activities := make([]activity, 0)
	warnings := make([]string, 0)
	for _, org := range orgs {
		items, orgWarnings := c.activities(org.Login, since, time.Now().UTC())
		activities = append(activities, items...)
		warnings = append(warnings, orgWarnings...)
	}
	sort.Slice(activities, func(i, j int) bool { return activities[i].CreatedAt.Before(activities[j].CreatedAt) })

	report := renderReport(since, time.Now().UTC(), orgs, activities, warnings)
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

func authToken() (string, error) {
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token, nil
	}
	command := exec.Command("gh", "auth", "token")
	output, err := command.Output()
	if err != nil {
		return "", errors.New("set GITHUB_TOKEN or authenticate with `gh auth login`")
	}
	if token := strings.TrimSpace(string(output)); token != "" {
		return token, nil
	}
	return "", errors.New("GitHub token is empty")
}

func (c *client) organizations() ([]organization, error) {
	var organizations []organization
	for page := 1; ; page++ {
		var batch []organization
		if err := c.get(fmt.Sprintf("/user/orgs?per_page=100&page=%d", page), &batch); err != nil {
			return nil, err
		}
		organizations = append(organizations, batch...)
		if len(batch) < 100 {
			return organizations, nil
		}
	}
}

func (c *client) activities(org string, since, until time.Time) ([]activity, []string) {
	activities := make([]activity, 0)
	warnings := make([]string, 0)
	events, err := c.eventActivities(org, since)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("%s events: %v", org, err))
	} else {
		activities = append(activities, events...)
	}
	commits, err := c.commitActivities(org, since, until)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("%s commits: %v", org, err))
	} else {
		activities = append(activities, commits...)
	}
	changes, err := c.issueActivities(org, since, until)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("%s issues/PRs: %v", org, err))
	} else {
		activities = append(activities, changes...)
	}
	return activities, warnings
}

func (c *client) eventActivities(org string, since time.Time) ([]activity, error) {
	activities := make([]activity, 0)
	for page := 1; page <= 10; page++ {
		var events []event
		path := fmt.Sprintf("/orgs/%s/events?per_page=100&page=%d", url.PathEscape(org), page)
		if err := c.get(path, &events); err != nil {
			return nil, err
		}
		if len(events) == 0 {
			break
		}
		for _, item := range events {
			if item.CreatedAt.Before(since) {
				return activities, nil
			}
			activities = append(activities, activity{
				Organization: org,
				Source:       "events",
				Type:         strings.TrimSuffix(item.Type, "Event"),
				Actor:        item.Actor.Login,
				Repository:   item.Repo.Name,
				CreatedAt:    item.CreatedAt,
				Summary:      summarize(item),
			})
		}
		if len(events) < 100 {
			break
		}
	}
	return activities, nil
}

func (c *client) commitActivities(org string, since, until time.Time) ([]activity, error) {
	activities := make([]activity, 0)
	for page := 1; page <= 10; page++ {
		query := url.Values{}
		query.Set("q", fmt.Sprintf("org:%s committer-date:%s..%s", org, since.Format(time.RFC3339), until.Format(time.RFC3339)))
		query.Set("per_page", "100")
		query.Set("page", fmt.Sprint(page))
		var result searchResponse[commitResult]
		if err := c.get("/search/commits?"+query.Encode(), &result); err != nil {
			return nil, err
		}
		for _, item := range result.Items {
			when := item.Commit.Committer.Date
			if when.IsZero() {
				when = item.Commit.Author.Date
			}
			if when.Before(since) || when.After(until) {
				continue
			}
			actor := item.Author.Login
			if actor == "" {
				actor = item.Commit.Committer.Name
			}
			if actor == "" {
				actor = item.Commit.Author.Name
			}
			activities = append(activities, activity{
				Organization: org,
				Source:       "commits",
				Type:         "Commit",
				Actor:        actor,
				Repository:   item.Repository.FullName,
				CreatedAt:    when,
				Summary:      firstLine(item.Commit.Message),
				URL:          item.HTMLURL,
			})
		}
		if len(result.Items) < 100 {
			break
		}
	}
	return activities, nil
}

func (c *client) issueActivities(org string, since, until time.Time) ([]activity, error) {
	activities := make([]activity, 0)
	for page := 1; page <= 10; page++ {
		query := url.Values{}
		query.Set("q", fmt.Sprintf("org:%s updated:%s..%s", org, since.Format(time.RFC3339), until.Format(time.RFC3339)))
		query.Set("per_page", "100")
		query.Set("page", fmt.Sprint(page))
		var result searchResponse[issueResult]
		if err := c.get("/search/issues?"+query.Encode(), &result); err != nil {
			return nil, err
		}
		for _, item := range result.Items {
			if item.UpdatedAt.Before(since) || item.UpdatedAt.After(until) {
				continue
			}
			typeName := "Issue"
			if item.PullRequest != nil {
				typeName = "Pull request"
			}
			repository := strings.TrimPrefix(item.RepositoryURL, "https://api.github.com/repos/")
			activities = append(activities, activity{
				Organization: org,
				Source:       "issues",
				Type:         typeName,
				Actor:        item.User.Login,
				Repository:   repository,
				CreatedAt:    item.UpdatedAt,
				Summary:      typeName + " updated: " + firstLine(item.Title),
				URL:          item.HTMLURL,
			})
		}
		if len(result.Items) < 100 {
			break
		}
	}
	return activities, nil
}

func (c *client) get(path string, target any) error {
	request, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("User-Agent", "bound-github-daily")
	request.Header.Set("Authorization", "Bearer "+c.token)
	response, err := c.http.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 1000))
		return fmt.Errorf("GitHub API %s: %s", response.Status, strings.TrimSpace(string(body)))
	}
	return json.NewDecoder(response.Body).Decode(target)
}

func summarize(item event) string {
	var payload map[string]any
	if json.Unmarshal(item.Payload, &payload) != nil {
		return item.Type
	}
	action, _ := payload["action"].(string)
	switch item.Type {
	case "PushEvent":
		if count, ok := payload["size"].(float64); ok {
			return fmt.Sprintf("pushed %.0f commit(s)", count)
		}
	case "PullRequestEvent", "IssuesEvent", "IssueCommentEvent", "PullRequestReviewEvent":
		if action != "" {
			return strings.TrimSuffix(item.Type, "Event") + " " + action
		}
	case "ReleaseEvent":
		if action != "" {
			return "Release " + action
		}
	}
	return strings.TrimSuffix(item.Type, "Event")
}

func firstLine(value string) string {
	return strings.TrimSpace(strings.SplitN(value, "\n", 2)[0])
}

func renderReport(since, until time.Time, orgs []organization, activities []activity, warnings []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# GitHub activity report\n\nPeriod: `%s` to `%s`\n\n", since.Format(time.RFC3339), until.Format(time.RFC3339))
	fmt.Fprintf(&b, "Organizations discovered: **%d**  \nActivities collected: **%d**\n\n", len(orgs), len(activities))
	byOrg := map[string][]activity{}
	bySource := map[string]int{}
	byActor := map[string]int{}
	systemActivities := 0
	for _, item := range activities {
		byOrg[item.Organization] = append(byOrg[item.Organization], item)
		bySource[item.Source]++
		if isHumanActor(item.Actor) {
			byActor[item.Actor]++
		} else {
			systemActivities++
		}
	}
	activeOrganizations := 0
	for _, org := range orgs {
		if len(byOrg[org.Login]) > 0 {
			activeOrganizations++
		}
	}
	b.WriteString("## Summary\n\n")
	fmt.Fprintf(&b, "**%d** activities across **%d** active organizations.\n\n", len(activities), activeOrganizations)
	b.WriteString("### By source\n\n")
	sources := make([]string, 0, len(bySource))
	for source := range bySource {
		sources = append(sources, source)
	}
	sort.Strings(sources)
	for _, source := range sources {
		fmt.Fprintf(&b, "- **%s:** %d\n", source, bySource[source])
	}
	actors := make([]string, 0, len(byActor))
	for actor := range byActor {
		actors = append(actors, actor)
	}
	sort.Slice(actors, func(i, j int) bool {
		if byActor[actors[i]] == byActor[actors[j]] {
			return actors[i] < actors[j]
		}
		return byActor[actors[i]] > byActor[actors[j]]
	})
	b.WriteString("\n### By user\n\n| User | Activities |\n|---|---:|\n")
	for _, actor := range actors {
		fmt.Fprintf(&b, "| %s | %d |\n", actor, byActor[actor])
	}
	if systemActivities > 0 {
		fmt.Fprintf(&b, "\n_%d automated/system activities omitted from this human-user table._\n", systemActivities)
	}
	b.WriteString("\n### By organization\n\n| Organization | Activities |\n|---|---:|\n")
	for _, org := range orgs {
		fmt.Fprintf(&b, "| %s | %d |\n", org.Login, len(byOrg[org.Login]))
	}
	b.WriteString("\n")
	if len(warnings) > 0 {
		b.WriteString("## Warnings\n\n")
		for _, warning := range warnings {
			fmt.Fprintf(&b, "- %s\n", warning)
		}
		b.WriteString("\n")
	}
	for _, org := range orgs {
		items := byOrg[org.Login]
		fmt.Fprintf(&b, "## %s (%d activities)\n\n", org.Login, len(items))
		if len(items) == 0 {
			b.WriteString("No activity found in the window.\n\n")
			continue
		}
		for _, item := range items {
			location := fmt.Sprintf("`%s` **%s** `%s`", item.CreatedAt.Format("15:04"), item.Actor, item.Repository)
			if item.URL != "" {
				location = fmt.Sprintf("[%s](%s)", location, item.URL)
			}
			fmt.Fprintf(&b, "- %s — _%s_ — %s\n", location, item.Source, item.Summary)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func isHumanActor(actor string) bool {
	name := strings.ToLower(strings.TrimSpace(actor))
	if name == "" || name == "github" || strings.HasSuffix(name, "[bot]") {
		return false
	}
	return !strings.Contains(name, "github-actions") && !strings.Contains(name, "dependabot")
}

func fatal(err error) { fmt.Fprintln(os.Stderr, "github-daily:", err); os.Exit(1) }

func checkArchitecture(path, sourceRoot string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open architecture: %w", err)
	}
	defer file.Close()
	architecture, err := parser.Parse(file)
	if err != nil {
		return fmt.Errorf("parse architecture: %w", err)
	}
	if err := architecture.Validate(); err != nil {
		return fmt.Errorf("validate architecture: %w", err)
	}
	if err := analyze.Go(sourceRoot, architecture); err != nil {
		return fmt.Errorf("check implementation: %w", err)
	}
	return nil
}
