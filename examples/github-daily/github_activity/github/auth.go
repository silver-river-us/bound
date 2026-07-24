package githubapi

import (
	"errors"
	"os"
	"os/exec"
	"strings"
)

func AuthToken() (string, error) {
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token, nil
	}
	command := exec.Command("gh", "auth", "token")
	output, err := command.Output()
	if err != nil {
		return "", errors.New("set GITHUB_TOKEN or authenticate with `gh auth login`")
	}
	if token := strings.TrimSpace(string(output)); token != "" {
		return token, nil
	}
	return "", errors.New("GitHub token is empty")
}
