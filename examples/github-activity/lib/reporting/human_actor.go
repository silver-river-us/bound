package reporting

import "strings"

func IsHumanActor(actor string) bool {
	name := strings.ToLower(strings.TrimSpace(actor))
	if name == "" || name == "github" || strings.HasSuffix(name, "[bot]") {
		return false
	}
	return !strings.Contains(name, "github-actions") && !strings.Contains(name, "dependabot")
}
