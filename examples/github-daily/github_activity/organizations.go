package githubactivity

import "fmt"

func (c *Client) Organizations() ([]Organization, error) {
	var organizations []Organization
	for page := 1; ; page++ {
		var batch []Organization
		if err := c.get(fmt.Sprintf("/user/orgs?per_page=100&page=%d", page), &batch); err != nil {
			return nil, err
		}
		organizations = append(organizations, batch...)
		if len(batch) < 100 {
			return organizations, nil
		}
	}
}
