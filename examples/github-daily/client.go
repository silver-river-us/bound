package main

import "net/http"

type client struct {
	baseURL string
	token   string
	http    *http.Client
}
