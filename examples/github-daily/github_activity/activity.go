package githubactivity

import "time"

type Activity struct {
	Organization string
	Source       string
	Type         string
	Actor        string
	Repository   string
	CreatedAt    time.Time
	Summary      string
	URL          string
}
