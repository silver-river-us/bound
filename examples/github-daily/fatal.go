package main

import (
	"fmt"
	"os"
)

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "github-daily:", err)
	os.Exit(1)
}
