package githubapi

import (
	"fmt"
	"time"

	"bound/examples/github-activity/lib/activity"
)

func (c *Client) Activities(org string, since, until time.Time) ([]githubactivity.Activity, []string) {
	activities := make([]githubactivity.Activity, 0)
	warnings := make([]string, 0)
	activities, warnings = appendSource(activities, warnings, org+" events", func() ([]githubactivity.Activity, error) { return c.eventActivities(org, since) })
	activities, warnings = appendSource(activities, warnings, org+" commits", func() ([]githubactivity.Activity, error) { return c.commitActivities(org, since, until) })
	activities, warnings = appendSource(activities, warnings, org+" issues/PRs", func() ([]githubactivity.Activity, error) { return c.issueActivities(org, since, until) })
	return activities, warnings
}

func appendSource(activities []githubactivity.Activity, warnings []string, label string, load func() ([]githubactivity.Activity, error)) ([]githubactivity.Activity, []string) {
	items, err := load()
	if err != nil {
		return activities, append(warnings, fmt.Sprintf("%s: %v", label, err))
	}
	return append(activities, items...), warnings
}
