package githubactivity

import (
	"fmt"
	"time"
)

func (c *Client) Activities(org string, since, until time.Time) ([]Activity, []string) {
	activities := make([]Activity, 0)
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
