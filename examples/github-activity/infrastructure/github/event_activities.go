package githubapi

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"bound/examples/github-activity/lib/activity"
)

func (c *Client) eventActivities(org string, since time.Time) ([]githubactivity.Activity, error) {
	activities := make([]githubactivity.Activity, 0)
	for page := 1; page <= 10; page++ {
		pageActivities, done, err := c.eventPageActivities(org, since, page)
		if err != nil {
			return nil, err
		}
		activities = append(activities, pageActivities...)
		if done {
			break
		}
	}
	return activities, nil
}

func (c *Client) eventPageActivities(org string, since time.Time, page int) ([]githubactivity.Activity, bool, error) {
	events, err := c.eventPage(org, page)
	if err != nil {
		return nil, false, err
	}
	if len(events) == 0 {
		return nil, true, nil
	}
	activities, beforeSince := appendEvents(nil, events, org, since)
	return activities, beforeSince || len(events) < 100, nil
}

func (c *Client) eventPage(org string, page int) ([]githubactivity.Event, error) {
	var events []githubactivity.Event
	path := fmt.Sprintf("/orgs/%s/events?per_page=100&page=%d", url.PathEscape(org), page)
	if err := c.get(path, &events); err != nil {
		return nil, err
	}
	return events, nil
}

func appendEvents(activities []githubactivity.Activity, events []githubactivity.Event, org string, since time.Time) ([]githubactivity.Activity, bool) {
	for _, item := range events {
		if item.CreatedAt.Before(since) {
			return activities, true
		}
		activities = append(activities, githubactivity.Activity{Organization: org, Source: "events", Type: strings.TrimSuffix(item.Type, "Event"), Actor: item.Actor.Login, Repository: item.Repo.Name, CreatedAt: item.CreatedAt, Summary: summarize(item)})
	}
	return activities, false
}
