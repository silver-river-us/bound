package main

type searchResponse[T any] struct {
	TotalCount int `json:"total_count"`
	Items      []T `json:"items"`
}
