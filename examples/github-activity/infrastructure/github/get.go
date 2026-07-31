package githubapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (c *Client) get(path string, target any) error {
	request, err := c.newRequest(path)
	if err != nil {
		return err
	}
	response, err := c.http.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if err := checkResponse(response); err != nil {
		return err
	}
	return json.NewDecoder(response.Body).Decode(target)
}

func (c *Client) newRequest(path string) (*http.Request, error) {
	request, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	request.Header.Set("User-Agent", "bound-github-activity")
	request.Header.Set("Authorization", "Bearer "+c.token)
	return request, nil
}

func checkResponse(response *http.Response) error {
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(response.Body, 1000))
	return fmt.Errorf("GitHub API %s: %s", response.Status, strings.TrimSpace(string(body)))
}
