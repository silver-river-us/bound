package cli

import "os"

func fatal(err error) {
	logger.Printf("error: %v", err)
	os.Exit(1)
}
