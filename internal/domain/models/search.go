package models

type SearchResult struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Labels []string `json:"labels"`
}
