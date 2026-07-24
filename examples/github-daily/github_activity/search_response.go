package githubactivity

type searchResponse[T any] struct {
	TotalCount int `json:"total_count"`
	Items      []T `json:"items"`
}
