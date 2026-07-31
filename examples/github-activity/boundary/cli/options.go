package cli

import "time"

type options struct {
	period       time.Duration
	output       string
	baseURL      string
	architecture string
	sourceRoot   string
}
