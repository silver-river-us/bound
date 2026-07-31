package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bound/internal/analyze"
	"bound/internal/parser"
	"bound/internal/render"
)

func main() {
	if len(os.Args) < 2 {
		fail("usage: bound <check|render|mermaid> [--root path] [-output path] <file.bo>")
	}
	command := os.Args[1]
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	root := flags.String("root", "", "source root to inspect with the Go backend")
	output := flags.String("output", "", "Markdown output path for the mermaid command")
	outputShort := flags.String("o", "", "short form of -output")
	if err := flags.Parse(os.Args[2:]); err != nil {
		fail(err.Error())
	}
	if flags.NArg() != 1 {
		fail("expected exactly one .bo file")
	}

	a, err := parser.ParseFile(flags.Arg(0))
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
	case "mermaid":
		outputPath := *output
		if outputPath == "" {
			outputPath = *outputShort
		}
		if outputPath == "" {
			outputPath = filepath.Join(filepath.Dir(flags.Arg(0)), strings.TrimSuffix(filepath.Base(flags.Arg(0)), filepath.Ext(flags.Arg(0)))+".md")
		}
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			fail(fmt.Sprintf("create Mermaid output directory: %v", err))
		}
		if err := os.WriteFile(outputPath, []byte(render.MermaidMarkdown(a)), 0o644); err != nil {
			fail(fmt.Sprintf("write Mermaid Markdown: %v", err))
		}
		fmt.Printf("wrote Mermaid Markdown to %s\n", outputPath)
	default:
		fail("unknown command: " + command)
	}
}

func fail(message string) { fmt.Fprintln(os.Stderr, "bound: "+message); os.Exit(1) }
