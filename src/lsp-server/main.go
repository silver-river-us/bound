package main

import (
	"fmt"
	"os"

	"bound/src/lsp"
)

func main() {
	if err := lsp.NewServer().Run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "bound-lsp:", err)
		os.Exit(1)
	}
}
