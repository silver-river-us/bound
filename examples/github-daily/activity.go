package main

import "time"

type activity struct {
	Organization string
	Source       string
	Type         string
	Actor        string
	Repository   string
	CreatedAt    time.Time
	Summary      string
	URL          string
}
