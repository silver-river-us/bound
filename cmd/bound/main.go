package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/silver-river-us/bound/internal/analyze"
	"github.com/silver-river-us/bound/internal/parser"
	"github.com/silver-river-us/bound/internal/render"
)

func main() {
	if len(os.Args) < 2 {
		fail("usage: bound <check|render> [--root path] <file.bo>")
	}
	command := os.Args[1]
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	root := flags.String("root", "", "source root to inspect with the Go backend")
	if err := flags.Parse(os.Args[2:]); err != nil {
		fail(err.Error())
	}
	if flags.NArg() != 1 {
		fail("expected exactly one .bo file")
	}

	file, err := os.Open(flags.Arg(0))
	if err != nil {
		fail(err.Error())
	}
	defer file.Close()
	a, err := parser.Parse(file)
	if err != nil {
		fail(err.Error())
	}
	if err := a.Validate(); err != nil {
		fail(err.Error())
	}

	switch command {
	case "check":
		if *root != "" {
			if err := analyze.Go(filepath.Clean(*root), a); err != nil {
				fail(err.Error())
			}
		}
		fmt.Printf("valid architecture %q (%d contexts, %d relationships)\n", a.Name, len(a.Contexts), len(a.Relations))
	case "render":
		fmt.Print(render.Structurizr(a))
	default:
		fail("unknown command: " + command)
	}
}

func fail(message string) { fmt.Fprintln(os.Stderr, "bound: "+message); os.Exit(1) }
