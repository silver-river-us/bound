package main

import "fmt"

func (c *client) organizations() ([]organization, error) {
	var organizations []organization
	for page := 1; ; page++ {
		var batch []organization
		if err := c.get(fmt.Sprintf("/user/orgs?per_page=100&page=%d", page), &batch); err != nil {
			return nil, err
		}
		organizations = append(organizations, batch...)
		if len(batch) < 100 {
			return organizations, nil
		}
	}
}
