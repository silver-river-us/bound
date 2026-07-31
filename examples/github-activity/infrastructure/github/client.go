package githubapi

import "net/http"

type Client struct {
	baseURL string
	token   string
	http    *http.Client
}
