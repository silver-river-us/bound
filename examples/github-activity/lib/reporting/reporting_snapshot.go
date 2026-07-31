package reporting

import "bound/examples/github-activity/lib/activity"

type ReportingSnapshot struct {
	Window        githubactivity.TimeWindow
	Organizations []githubactivity.Organization
	Feed          githubactivity.ActivityFeed
}
