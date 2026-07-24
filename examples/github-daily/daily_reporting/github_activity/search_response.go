package githubactivity

type SearchResponse[T any] struct {
	TotalCount int `json:"total_count"`
	Items      []T `json:"items"`
}
