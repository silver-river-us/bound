package main

import (
	"fmt"
	"github.com/silver-river-us/bound/internal/parser"
	"github.com/silver-river-us/bound/internal/render"
	"os"
)

func main() {
	if len(os.Args) < 3 {
		fail("usage: bound <check|render> <file.bo>")
	}
	file, err := os.Open(os.Args[2])
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
	switch os.Args[1] {
	case "check":
		fmt.Printf("valid architecture %q (%d contexts, %d relationships)\n", a.Name, len(a.Contexts), len(a.Relations))
	case "render":
		fmt.Print(render.Structurizr(a))
	default:
		fail("unknown command: " + os.Args[1])
	}
}

func fail(message string) { fmt.Fprintln(os.Stderr, "bound: "+message); os.Exit(1) }
