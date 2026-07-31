// Package reporting coordinates the daily activity use case.
package reporting

import (
	"sort"
	"time"

	githubactivity "bound/examples/github-activity/lib/activity"
)

func Generate(source githubactivity.Source, since, until time.Time) Result {
	organizations, err := source.Organizations()
	if err != nil {
		return Result{Err: err}
	}
	activities, warnings := collect(source, organizations, since, until)
	sort.Slice(activities, func(i, j int) bool { return activities[i].CreatedAt.Before(activities[j].CreatedAt) })
	snapshot := ReportingSnapshot{
		Window:        githubactivity.TimeWindow{Since: since, Until: until},
		Organizations: organizations,
		Feed:          githubactivity.ActivityFeed{Activities: activities, Warnings: warnings},
	}
	return buildResult(snapshot)
}

func buildResult(snapshot ReportingSnapshot) Result {
	return Result{Report: Report{Snapshot: snapshot, Summary: summarize(snapshot)}, ActivityCount: len(snapshot.Feed.Activities), OrganizationCount: len(snapshot.Organizations)}
}

func collect(source githubactivity.Source, organizations []githubactivity.Organization, since, until time.Time) ([]githubactivity.Activity, []string) {
	activities := make([]githubactivity.Activity, 0)
	warnings := make([]string, 0)
	for _, organization := range organizations {
		items, organizationWarnings := source.Activities(organization.Login, since, until)
		activities = append(activities, items...)
		warnings = append(warnings, organizationWarnings...)
	}
	return activities, warnings
}
