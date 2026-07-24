package githubapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/silver-river-us/bound/examples/github-daily/daily_reporting/activity"
)

func summarize(item githubactivity.Event) string {
	var payload map[string]any
	if json.Unmarshal(item.Payload, &payload) != nil {
		return item.Type
	}
	action, _ := payload["action"].(string)
	switch item.Type {
	case "PushEvent":
		if count, ok := payload["size"].(float64); ok {
			return fmt.Sprintf("pushed %.0f commit(s)", count)
		}
	case "PullRequestEvent", "IssuesEvent", "IssueCommentEvent", "PullRequestReviewEvent":
		if action != "" {
			return strings.TrimSuffix(item.Type, "Event") + " " + action
		}
	case "ReleaseEvent":
		if action != "" {
			return "Release " + action
		}
	}
	return strings.TrimSuffix(item.Type, "Event")
}
