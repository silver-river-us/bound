package reporting

import "bound/examples/github-activity/lib/activity"

func summarize(snapshot ReportingSnapshot) Summary {
	summary := Summary{
		ActivitiesByOrganization: map[string][]githubactivity.Activity{},
		ActivitiesBySource:       map[string]int{},
		ActivitiesByActor:        map[string]int{},
	}
	addActivityStatistics(&summary, snapshot.Feed.Activities)
	summary.ActiveOrganizations = countActiveOrganizations(summary.ActivitiesByOrganization, snapshot.Organizations)
	return summary
}

func addActivityStatistics(summary *Summary, activities []githubactivity.Activity) {
	for _, item := range activities {
		summary.ActivitiesByOrganization[item.Organization] = append(summary.ActivitiesByOrganization[item.Organization], item)
		summary.ActivitiesBySource[item.Source]++
		if IsHumanActor(item.Actor) {
			summary.ActivitiesByActor[item.Actor]++
		} else {
			summary.SystemActivities++
		}
	}
}

func countActiveOrganizations(byOrganization map[string][]githubactivity.Activity, organizations []githubactivity.Organization) int {
	active := 0
	for _, organization := range organizations {
		if len(byOrganization[organization.Login]) > 0 {
			active++
		}
	}
	return active
}
