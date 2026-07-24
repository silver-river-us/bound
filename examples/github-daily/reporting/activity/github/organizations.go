package githubapi

import (
	"fmt"

	"github.com/silver-river-us/bound/examples/github-daily/reporting/activity"
)

func (c *Client) Organizations() ([]githubactivity.Organization, error) {
	var organizations []githubactivity.Organization
	for page := 1; ; page++ {
		var batch []githubactivity.Organization
		if err := c.get(fmt.Sprintf("/user/orgs?per_page=100&page=%d", page), &batch); err != nil {
			return nil, err
		}
		organizations = append(organizations, batch...)
		if len(batch) < 100 {
			return organizations, nil
		}
	}
}
