package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"bound/src/compiler"
	"bound/src/render"
)

func main() {
	if len(os.Args) < 2 {
		fail("usage: bound <compile|review> <file.bo>")
	}
	command := os.Args[1]
	if len(os.Args) != 3 {
		fail("expected exactly one .bo file")
	}
	architecturePath := os.Args[2]

	switch command {
	case "compile":
		program, err := compiler.Compile(architecturePath, compiler.Options{})
		if err != nil {
			fail(err.Error())
		}
		encoded, err := program.JSON()
		if err != nil {
			fail(fmt.Sprintf("encode compiler IR: %v", err))
		}
		fmt.Println(string(encoded))
	case "review":
		program, err := compiler.Compile(architecturePath, compiler.Options{})
		if err != nil {
			fail(err.Error())
		}
		outputPath := filepath.Join(filepath.Dir(architecturePath), strings.TrimSuffix(filepath.Base(architecturePath), filepath.Ext(architecturePath))+".html")
		if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
			fail(fmt.Sprintf("create HTML output directory: %v", err))
		}
		if err := os.WriteFile(outputPath, []byte(render.MermaidHTML(program.Architecture)), 0o644); err != nil {
			fail(fmt.Sprintf("write HTML architecture review: %v", err))
		}
		if err := openReview(outputPath); err != nil {
			fail(fmt.Sprintf("open HTML architecture review: %v (file written to %s)", err, outputPath))
		}
		fmt.Printf("opened HTML architecture review %s\n", outputPath)
	default:
		fail("unknown command: " + command)
	}
}

func openReview(path string) error {
	var command string
	var arguments []string
	switch runtime.GOOS {
	case "darwin":
		command = "open"
		arguments = []string{path}
	case "windows":
		command = "cmd"
		arguments = []string{"/c", "start", "", path}
	default:
		command = "xdg-open"
		arguments = []string{path}
	}
	return exec.Command(command, arguments...).Start()
}

func fail(message string) { fmt.Fprintln(os.Stderr, "bound: "+message); os.Exit(1) }
