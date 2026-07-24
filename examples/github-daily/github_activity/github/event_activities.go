package githubapi

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/silver-river-us/bound/examples/github-daily/github_activity"
)

func (c *Client) eventActivities(org string, since time.Time) ([]githubactivity.Activity, error) {
	activities := make([]githubactivity.Activity, 0)
	for page := 1; page <= 10; page++ {
		var events []githubactivity.Event
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
			activities = append(activities, githubactivity.Activity{
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
