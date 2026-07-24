package githubactivity

import "time"

type TimeWindow struct {
	Since time.Time
	Until time.Time
}

type ActivityFeed struct {
	Activities []Activity
	Warnings   []string
}
