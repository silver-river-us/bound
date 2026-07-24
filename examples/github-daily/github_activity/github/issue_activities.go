package githubapi

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/silver-river-us/bound/examples/github-daily/github_activity"
)

func (c *Client) issueActivities(org string, since, until time.Time) ([]githubactivity.Activity, error) {
	activities := make([]githubactivity.Activity, 0)
	for page := 1; page <= 10; page++ {
		query := url.Values{}
		query.Set("q", fmt.Sprintf("org:%s updated:%s..%s", org, since.Format(time.RFC3339), until.Format(time.RFC3339)))
		query.Set("per_page", "100")
		query.Set("page", fmt.Sprint(page))
		var result githubactivity.SearchResponse[githubactivity.IssueResult]
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
			activities = append(activities, githubactivity.Activity{
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
