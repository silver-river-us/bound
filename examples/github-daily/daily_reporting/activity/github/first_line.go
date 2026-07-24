package githubapi

import "strings"

func firstLine(value string) string {
	return strings.TrimSpace(strings.SplitN(value, "\n", 2)[0])
}
