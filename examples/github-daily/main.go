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
	Type         string
	Actor        string
	Repository   string
	CreatedAt    time.Time
	Summary      string
}

func main() {
	defaultOutput := filepath.Join("reports", "github-activity-"+time.Now().UTC().Format("2006-01-02")+".md")
	sinceFlag := flag.Duration("since", 24*time.Hour, "report window, for example 24h or 48h")
	outputFlag := flag.String("output", defaultOutput, "Markdown report path; use - for stdout")
	baseURLFlag := flag.String("base-url", githubAPI, "GitHub API base URL")
	flag.Parse()

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
		items, err := c.activities(org.Login, since)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", org.Login, err))
			continue
		}
		activities = append(activities, items...)
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

func (c *client) activities(org string, since time.Time) ([]activity, error) {
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

func renderReport(since, until time.Time, orgs []organization, activities []activity, warnings []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# GitHub activity report\n\nPeriod: `%s` to `%s`\n\n", since.Format(time.RFC3339), until.Format(time.RFC3339))
	fmt.Fprintf(&b, "Organizations discovered: **%d**  \nActivities collected: **%d**\n\n", len(orgs), len(activities))
	if len(warnings) > 0 {
		b.WriteString("## Warnings\n\n")
		for _, warning := range warnings {
			fmt.Fprintf(&b, "- %s\n", warning)
		}
		b.WriteString("\n")
	}
	byOrg := map[string][]activity{}
	for _, item := range activities {
		byOrg[item.Organization] = append(byOrg[item.Organization], item)
	}
	for _, org := range orgs {
		items := byOrg[org.Login]
		fmt.Fprintf(&b, "## %s (%d activities)\n\n", org.Login, len(items))
		if len(items) == 0 {
			b.WriteString("No activity found in the window.\n\n")
			continue
		}
		for _, item := range items {
			fmt.Fprintf(&b, "- `%s` **%s** `%s` — %s\n", item.CreatedAt.Format("15:04"), item.Actor, item.Repository, item.Summary)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func fatal(err error) { fmt.Fprintln(os.Stderr, "github-daily:", err); os.Exit(1) }
