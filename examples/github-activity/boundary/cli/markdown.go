package cli

import (
	"fmt"
	"sort"
	"strings"

	githubactivity "bound/examples/github-activity/lib/activity"
	"bound/examples/github-activity/lib/reporting"
)

func RenderMarkdown(report reporting.Report) string {
	var builder strings.Builder
	writeHeader(&builder, report)
	writeSummary(&builder, report)
	writeWarnings(&builder, report.Snapshot.Feed.Warnings)
	writeOrganizations(&builder, report)
	return builder.String()
}

func writeHeader(builder *strings.Builder, report reporting.Report) {
	snapshot := report.Snapshot
	fmt.Fprintf(builder, "# GitHub activity report\n\nPeriod: `%s` to `%s`\n\n", snapshot.Window.Since.Format("2006-01-02T15:04:05Z07:00"), snapshot.Window.Until.Format("2006-01-02T15:04:05Z07:00"))
	fmt.Fprintf(builder, "Organizations discovered: **%d**  \nActivities collected: **%d**\n\n", len(snapshot.Organizations), len(snapshot.Feed.Activities))
}

func writeSummary(builder *strings.Builder, report reporting.Report) {
	snapshot, summary := report.Snapshot, report.Summary
	builder.WriteString("## Summary\n\n")
	fmt.Fprintf(builder, "**%d** activities across **%d** active organizations.\n\n", len(snapshot.Feed.Activities), summary.ActiveOrganizations)
	writeSources(builder, summary.ActivitiesBySource)
	writeActors(builder, summary.ActivitiesByActor, summary.SystemActivities)
	writeOrganizationSummary(builder, summary.ActivitiesByOrganization, snapshot.Organizations)
}

func writeSources(builder *strings.Builder, bySource map[string]int) {
	builder.WriteString("### By source\n\n")
	sources := make([]string, 0, len(bySource))
	for source := range bySource {
		sources = append(sources, source)
	}
	sort.Strings(sources)
	for _, source := range sources {
		fmt.Fprintf(builder, "- **%s:** %d\n", source, bySource[source])
	}
}

func writeActors(builder *strings.Builder, byActor map[string]int, systemActivities int) {
	actors := sortedActors(byActor)
	builder.WriteString("\n### By user\n\n| User | Activities |\n|---|---:|\n")
	for _, actor := range actors {
		fmt.Fprintf(builder, "| %s | %d |\n", actor, byActor[actor])
	}
	if systemActivities > 0 {
		fmt.Fprintf(builder, "\n_%d automated/system activities omitted from this human-user table._\n", systemActivities)
	}
}

func sortedActors(byActor map[string]int) []string {
	actors := make([]string, 0, len(byActor))
	for actor := range byActor {
		actors = append(actors, actor)
	}
	sort.Slice(actors, func(i, j int) bool {
		if byActor[actors[i]] == byActor[actors[j]] {
			return actors[i] < actors[j]
		}
		return byActor[actors[i]] > byActor[actors[j]]
	})
	return actors
}

func writeOrganizationSummary(builder *strings.Builder, byOrganization map[string][]githubactivity.Activity, organizations []githubactivity.Organization) {
	builder.WriteString("\n### By organization\n\n| Organization | Activities |\n|---|---:|\n")
	for _, organization := range organizations {
		fmt.Fprintf(builder, "| %s | %d |\n", organization.Login, len(byOrganization[organization.Login]))
	}
	builder.WriteString("\n")
}

func writeWarnings(builder *strings.Builder, warnings []string) {
	if len(warnings) == 0 {
		return
	}
	builder.WriteString("## Warnings\n\n")
	for _, warning := range warnings {
		fmt.Fprintf(builder, "- %s\n", warning)
	}
	builder.WriteString("\n")
}

func writeOrganizations(builder *strings.Builder, report reporting.Report) {
	for _, organization := range report.Snapshot.Organizations {
		writeOrganization(builder, organization, report.Summary.ActivitiesByOrganization[organization.Login])
	}
}

func writeOrganization(builder *strings.Builder, organization githubactivity.Organization, activities []githubactivity.Activity) {
	fmt.Fprintf(builder, "## %s (%d activities)\n\n", organization.Login, len(activities))
	if len(activities) == 0 {
		builder.WriteString("No activity found in the window.\n\n")
		return
	}
	for _, activity := range activities {
		writeActivity(builder, activity)
	}
	builder.WriteString("\n")
}

func writeActivity(builder *strings.Builder, activity githubactivity.Activity) {
	location := fmt.Sprintf("`%s` **%s** `%s`", activity.CreatedAt.Format("15:04"), activity.Actor, activity.Repository)
	if activity.URL != "" {
		location = fmt.Sprintf("[%s](%s)", location, activity.URL)
	}
	fmt.Fprintf(builder, "- %s — _%s_ — %s\n", location, activity.Source, activity.Summary)
}
