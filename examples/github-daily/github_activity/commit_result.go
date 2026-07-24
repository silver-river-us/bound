package githubactivity

import "time"

type CommitResult struct {
	HTMLURL string `json:"html_url"`
	Author  struct {
		Login string `json:"login"`
	} `json:"author"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	Commit struct {
		Message string `json:"message"`
		Author  struct {
			Name string    `json:"name"`
			Date time.Time `json:"date"`
		} `json:"author"`
		Committer struct {
			Name string    `json:"name"`
			Date time.Time `json:"date"`
		} `json:"committer"`
	} `json:"commit"`
}
