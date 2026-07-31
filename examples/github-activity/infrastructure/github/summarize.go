package githubapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"bound/examples/github-activity/lib/activity"
)

func summarize(item githubactivity.Event) string {
	var payload map[string]any
	if json.Unmarshal(item.Payload, &payload) != nil {
		return item.Type
	}
	return summarizePayload(item.Type, payload)
}

func summarizePayload(eventType string, payload map[string]any) string {
	action, _ := payload["action"].(string)
	return summarizeEvent(eventType, action, payload["size"])
}

func summarizeEvent(eventType, action string, size any) string {
	if eventType == "PushEvent" {
		if count, ok := size.(float64); ok {
			return fmt.Sprintf("pushed %.0f commit(s)", count)
		}
	}
	if summary, ok := summarizeAction(eventType, action); ok {
		return summary
	}
	return strings.TrimSuffix(eventType, "Event")
}

func summarizeAction(eventType, action string) (string, bool) {
	if action == "" {
		return "", false
	}
	switch eventType {
	case "PullRequestEvent", "IssuesEvent", "IssueCommentEvent", "PullRequestReviewEvent":
		return strings.TrimSuffix(eventType, "Event") + " " + action, true
	case "ReleaseEvent":
		return "Release " + action, true
	default:
		return "", false
	}
}
