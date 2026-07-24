package githubactivity

import (
	"fmt"
	"net/url"
	"time"
)

func (c *Client) commitActivities(org string, since, until time.Time) ([]Activity, error) {
	activities := make([]Activity, 0)
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
			activities = append(activities, Activity{
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
