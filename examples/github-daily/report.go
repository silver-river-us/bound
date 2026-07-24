package main

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

func renderReport(since, until time.Time, orgs []organization, activities []activity, warnings []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# GitHub activity report\n\nPeriod: `%s` to `%s`\n\n", since.Format(time.RFC3339), until.Format(time.RFC3339))
	fmt.Fprintf(&b, "Organizations discovered: **%d**  \nActivities collected: **%d**\n\n", len(orgs), len(activities))
	byOrg := map[string][]activity{}
	bySource := map[string]int{}
	byActor := map[string]int{}
	systemActivities := 0
	for _, item := range activities {
		byOrg[item.Organization] = append(byOrg[item.Organization], item)
		bySource[item.Source]++
		if isHumanActor(item.Actor) {
			byActor[item.Actor]++
		} else {
			systemActivities++
		}
	}
	activeOrganizations := 0
	for _, org := range orgs {
		if len(byOrg[org.Login]) > 0 {
			activeOrganizations++
		}
	}
	b.WriteString("## Summary\n\n")
	fmt.Fprintf(&b, "**%d** activities across **%d** active organizations.\n\n", len(activities), activeOrganizations)
	b.WriteString("### By source\n\n")
	sources := make([]string, 0, len(bySource))
	for source := range bySource {
		sources = append(sources, source)
	}
	sort.Strings(sources)
	for _, source := range sources {
		fmt.Fprintf(&b, "- **%s:** %d\n", source, bySource[source])
	}
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
	b.WriteString("\n### By user\n\n| User | Activities |\n|---|---:|\n")
	for _, actor := range actors {
		fmt.Fprintf(&b, "| %s | %d |\n", actor, byActor[actor])
	}
	if systemActivities > 0 {
		fmt.Fprintf(&b, "\n_%d automated/system activities omitted from this human-user table._\n", systemActivities)
	}
	b.WriteString("\n### By organization\n\n| Organization | Activities |\n|---|---:|\n")
	for _, org := range orgs {
		fmt.Fprintf(&b, "| %s | %d |\n", org.Login, len(byOrg[org.Login]))
	}
	b.WriteString("\n")
	if len(warnings) > 0 {
		b.WriteString("## Warnings\n\n")
		for _, warning := range warnings {
			fmt.Fprintf(&b, "- %s\n", warning)
		}
		b.WriteString("\n")
	}
	for _, org := range orgs {
		items := byOrg[org.Login]
		fmt.Fprintf(&b, "## %s (%d activities)\n\n", org.Login, len(items))
		if len(items) == 0 {
			b.WriteString("No activity found in the window.\n\n")
			continue
		}
		for _, item := range items {
			location := fmt.Sprintf("`%s` **%s** `%s`", item.CreatedAt.Format("15:04"), item.Actor, item.Repository)
			if item.URL != "" {
				location = fmt.Sprintf("[%s](%s)", location, item.URL)
			}
			fmt.Fprintf(&b, "- %s — _%s_ — %s\n", location, item.Source, item.Summary)
		}
		b.WriteString("\n")
	}
	return b.String()
}
