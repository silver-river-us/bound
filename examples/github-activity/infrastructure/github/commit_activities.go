package githubapi

import (
	"fmt"
	"net/url"
	"time"

	"bound/examples/github-activity/lib/activity"
)

func (c *Client) commitActivities(org string, since, until time.Time) ([]githubactivity.Activity, error) {
	activities := make([]githubactivity.Activity, 0)
	for page := 1; page <= 10; page++ {
		result, err := c.commitPage(org, since, until, page)
		if err != nil {
			return nil, err
		}
		activities = appendCommits(activities, result.Items, org, since, until)
		if len(result.Items) < 100 {
			break
		}
	}
	return activities, nil
}

func (c *Client) commitPage(org string, since, until time.Time, page int) (githubactivity.SearchResponse[githubactivity.CommitResult], error) {
	query := url.Values{}
	query.Set("q", fmt.Sprintf("org:%s committer-date:%s..%s", org, since.Format(time.RFC3339), until.Format(time.RFC3339)))
	query.Set("per_page", "100")
	query.Set("page", fmt.Sprint(page))
	var result githubactivity.SearchResponse[githubactivity.CommitResult]
	if err := c.get("/search/commits?"+query.Encode(), &result); err != nil {
		return result, err
	}
	return result, nil
}

func appendCommits(activities []githubactivity.Activity, items []githubactivity.CommitResult, org string, since, until time.Time) []githubactivity.Activity {
	for _, item := range items {
		activity, ok := commitActivity(item, org, since, until)
		if ok {
			activities = append(activities, activity)
		}
	}
	return activities
}

func commitActivity(item githubactivity.CommitResult, org string, since, until time.Time) (githubactivity.Activity, bool) {
	when := commitTime(item)
	if when.Before(since) || when.After(until) {
		return githubactivity.Activity{}, false
	}
	actor := commitActor(item)
	return githubactivity.Activity{Organization: org, Source: "commits", Type: "Commit", Actor: actor, Repository: item.Repository.FullName, CreatedAt: when, Summary: firstLine(item.Commit.Message), URL: item.HTMLURL}, true
}

func commitTime(item githubactivity.CommitResult) time.Time {
	if !item.Commit.Committer.Date.IsZero() {
		return item.Commit.Committer.Date
	}
	return item.Commit.Author.Date
}

func commitActor(item githubactivity.CommitResult) string {
	if item.Author.Login != "" {
		return item.Author.Login
	}
	if item.Commit.Committer.Name != "" {
		return item.Commit.Committer.Name
	}
	return item.Commit.Author.Name
}
