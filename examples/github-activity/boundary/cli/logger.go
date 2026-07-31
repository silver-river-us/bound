package cli

import (
	"log"
	"os"
)

var logger = log.New(os.Stderr, "github-activity: ", log.LstdFlags)
