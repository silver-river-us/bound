package githubactivity

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

func (c *Client) eventActivities(org string, since time.Time) ([]Activity, error) {
	activities := make([]Activity, 0)
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
			activities = append(activities, Activity{
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
