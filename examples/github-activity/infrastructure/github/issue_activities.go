package githubapi

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"bound/examples/github-activity/lib/activity"
)

func (c *Client) issueActivities(org string, since, until time.Time) ([]githubactivity.Activity, error) {
	activities := make([]githubactivity.Activity, 0)
	for page := 1; page <= 10; page++ {
		result, err := c.issuePage(org, since, until, page)
		if err != nil {
			return nil, err
		}
		activities = appendIssues(activities, result.Items, org, since, until)
		if len(result.Items) < 100 {
			break
		}
	}
	return activities, nil
}

func (c *Client) issuePage(org string, since, until time.Time, page int) (githubactivity.SearchResponse[githubactivity.IssueResult], error) {
	query := url.Values{}
	query.Set("q", fmt.Sprintf("org:%s updated:%s..%s", org, since.Format(time.RFC3339), until.Format(time.RFC3339)))
	query.Set("per_page", "100")
	query.Set("page", fmt.Sprint(page))
	var result githubactivity.SearchResponse[githubactivity.IssueResult]
	if err := c.get("/search/issues?"+query.Encode(), &result); err != nil {
		return result, err
	}
	return result, nil
}

func appendIssues(activities []githubactivity.Activity, items []githubactivity.IssueResult, org string, since, until time.Time) []githubactivity.Activity {
	for _, item := range items {
		if activity, ok := issueActivity(item, org, since, until); ok {
			activities = append(activities, activity)
		}
	}
	return activities
}

func issueActivity(item githubactivity.IssueResult, org string, since, until time.Time) (githubactivity.Activity, bool) {
	if item.UpdatedAt.Before(since) || item.UpdatedAt.After(until) {
		return githubactivity.Activity{}, false
	}
	typeName := "Issue"
	if item.PullRequest != nil {
		typeName = "Pull request"
	}
	repository := strings.TrimPrefix(item.RepositoryURL, "https://api.github.com/repos/")
	return githubactivity.Activity{Organization: org, Source: "issues", Type: typeName, Actor: item.User.Login, Repository: repository, CreatedAt: item.UpdatedAt, Summary: typeName + " updated: " + firstLine(item.Title), URL: item.HTMLURL}, true
}
