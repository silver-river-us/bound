package reporting

import "bound/examples/github-activity/lib/activity"

type Summary struct {
	ActivitiesByOrganization map[string][]githubactivity.Activity
	ActivitiesBySource       map[string]int
	ActivitiesByActor        map[string]int
	ActiveOrganizations      int
	SystemActivities         int
}
