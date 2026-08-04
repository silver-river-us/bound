package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"bound/src/boundary/render"
	"bound/src/lib/compiler"
	"bound/src/lib/format"
	"bound/src/lib/model"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "compile":
		runCompile(os.Args[2:])
	case "check":
		runCheck(os.Args[2:])
	case "fmt":
		runFormat(os.Args[2:])
	case "init":
		runInit(os.Args[2:])
	case "review":
		runReview(os.Args[2:])
	case "doctor":
		runDoctor(os.Args[2:])
	case "diff":
		runDiff(os.Args[2:])
	case "version", "--version", "-v":
		fmt.Println(version)
	case "help", "--help", "-h":
		usage()
	default:
		fail("unknown command: " + os.Args[1])
	}
}

func runCompile(arguments []string) {
	arguments = moveFlagsBeforePath(arguments, "--design-only")
	flags := flag.NewFlagSet("compile", flag.ExitOnError)
	designOnly := flags.Bool("design-only", false, "validate the model without inspecting implementation files")
	flags.Usage = func() { fail("usage: bound compile [--design-only] <file.bo>") }
	flags.Parse(arguments)
	path := onePath(flags.Args(), "compile")
	program, err := compiler.Compile(path, compiler.Options{SkipImplementation: *designOnly})
	if err != nil {
		failError(err, false)
	}
	encoded, err := program.JSON()
	if err != nil {
		fail("encode compiler IR: " + err.Error())
	}
	fmt.Println(string(encoded))
}

func runCheck(arguments []string) {
	arguments = moveFlagsBeforePath(arguments, "--design-only", "--watch", "--json")
	flags := flag.NewFlagSet("check", flag.ExitOnError)
	designOnly := flags.Bool("design-only", false, "validate the model without inspecting implementation files")
	watch := flags.Bool("watch", false, "rerun whenever the architecture file changes")
	jsonOutput := flags.Bool("json", false, "emit machine-readable diagnostics")
	flags.Usage = func() { fail("usage: bound check [--design-only] [--watch] [--json] <file.bo>") }
	flags.Parse(arguments)
	path := onePath(flags.Args(), "check")
	if *watch {
		watchCheck(path, *designOnly, *jsonOutput)
		return
	}
	if err := check(path, *designOnly, *jsonOutput); err != nil {
		os.Exit(1)
	}
}

func check(path string, designOnly, jsonOutput bool) error {
	_, err := compiler.Compile(path, compiler.Options{SkipImplementation: designOnly})
	if err != nil {
		if jsonOutput {
			printDiagnostics(err)
		} else {
			fmt.Fprintln(os.Stderr, "bound: "+err.Error())
		}
		return err
	}
	if jsonOutput {
		fmt.Println(`{"ok":true}`)
	} else {
		fmt.Printf("ok: %s\n", path)
	}
	return nil
}

func watchCheck(path string, designOnly, jsonOutput bool) {
	var last time.Time
	for {
		info, err := os.Stat(path)
		if err != nil {
			fail("watch " + path + ": " + err.Error())
		}
		if info.ModTime().After(last) {
			last = info.ModTime()
			if !jsonOutput {
				fmt.Printf("checking %s\n", path)
			}
			_ = check(path, designOnly, jsonOutput)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func runFormat(arguments []string) {
	arguments = moveFlagsBeforePath(arguments, "-w", "--check")
	flags := flag.NewFlagSet("fmt", flag.ExitOnError)
	write := flags.Bool("w", false, "write the formatted source back to the file")
	checkOnly := flags.Bool("check", false, "fail if the file is not canonically formatted")
	flags.Usage = func() { fail("usage: bound fmt [-w|--check] <file.bo>") }
	flags.Parse(arguments)
	path := onePath(flags.Args(), "fmt")
	content, err := os.ReadFile(path)
	if err != nil {
		fail("read " + path + ": " + err.Error())
	}
	formatted, err := format.Format(string(content))
	if err != nil {
		fail("format " + path + ": " + err.Error())
	}
	if *checkOnly {
		if string(content) != formatted {
			fmt.Fprintln(os.Stderr, "bound: file is not formatted: "+path)
			os.Exit(1)
		}
		return
	}
	if *write {
		if err := os.WriteFile(path, []byte(formatted), 0o644); err != nil {
			fail("write " + path + ": " + err.Error())
		}
		return
	}
	fmt.Print(formatted)
}

func runInit(arguments []string) {
	arguments = moveFlagsBeforePath(arguments, "--force")
	flags := flag.NewFlagSet("init", flag.ExitOnError)
	force := flags.Bool("force", false, "replace an existing architecture.bo")
	flags.Usage = func() { fail("usage: bound init [--force] [directory]") }
	flags.Parse(arguments)
	directory := "."
	if len(flags.Args()) > 1 {
		fail("usage: bound init [--force] [directory]")
	}
	if len(flags.Args()) == 1 {
		directory = flags.Args()[0]
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		fail("create project directory: " + err.Error())
	}
	path := filepath.Join(directory, "architecture.bo")
	if _, err := os.Stat(path); err == nil && !*force {
		fail(path + " already exists (use --force to replace it)")
	}
	content := `"""
Describe the architecture and its boundaries here.
"""
architecture Example do
  implementation go "."

  context Boundary do
  end
end
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		fail("write " + path + ": " + err.Error())
	}
	fmt.Printf("created %s\n", path)
	fmt.Println("next: bound fmt -w " + path)
	fmt.Println("next: bound check --design-only " + path)
}

func runDiff(arguments []string) {
	arguments = moveFlagsBeforePath(arguments, "--json")
	flags := flag.NewFlagSet("diff", flag.ExitOnError)
	jsonOutput := flags.Bool("json", false, "emit machine-readable results")
	flags.Usage = func() { fail("usage: bound diff [--json] <before.bo> <after.bo>") }
	flags.Parse(arguments)
	if len(flags.Args()) != 2 {
		fail("usage: bound diff [--json] <before.bo> <after.bo>")
	}
	before, err := compiler.Compile(flags.Args()[0], compiler.Options{SkipImplementation: true})
	if err != nil {
		failError(err, *jsonOutput)
	}
	after, err := compiler.Compile(flags.Args()[1], compiler.Options{SkipImplementation: true})
	if err != nil {
		failError(err, *jsonOutput)
	}
	result := architectureDiff(before.Architecture, after.Architecture)
	if *jsonOutput {
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			fail("encode architecture diff: " + err.Error())
		}
		return
	}
	fmt.Printf("Architecture diff: %s -> %s\n", flags.Args()[0], flags.Args()[1])
	printDiffSection("Contexts added", result.Contexts.Added)
	printDiffSection("Contexts removed", result.Contexts.Removed)
	printDiffSection("Contexts changed", result.Contexts.Changed)
	printDiffSection("Modules added", result.Modules.Added)
	printDiffSection("Modules removed", result.Modules.Removed)
	printDiffSection("Modules changed", result.Modules.Changed)
	printDiffSection("Relationships added", result.Relationships.Added)
	printDiffSection("Relationships removed", result.Relationships.Removed)
}

type diffSection struct {
	Added   []string `json:"added,omitempty"`
	Removed []string `json:"removed,omitempty"`
	Changed []string `json:"changed,omitempty"`
}

type architectureDiffResult struct {
	Contexts      diffSection `json:"contexts"`
	Modules       diffSection `json:"modules"`
	Relationships diffSection `json:"relationships"`
}

func architectureDiff(before, after *model.Architecture) architectureDiffResult {
	result := architectureDiffResult{
		Contexts: diffSection{Added: mapDifference(after.Contexts, before.Contexts), Removed: mapDifference(before.Contexts, after.Contexts)},
		Modules:  diffSection{Added: mapDifference(after.Modules, before.Modules), Removed: mapDifference(before.Modules, after.Modules)},
	}
	for name, oldValue := range before.Contexts {
		newValue, exists := after.Contexts[name]
		if exists && jsonValue(oldValue) != jsonValue(newValue) {
			result.Contexts.Changed = append(result.Contexts.Changed, name)
		}
	}
	for name, oldValue := range before.Modules {
		newValue, exists := after.Modules[name]
		if exists && jsonValue(oldValue) != jsonValue(newValue) {
			result.Modules.Changed = append(result.Modules.Changed, name)
		}
	}
	beforeRelations := relationNames(before)
	afterRelations := relationNames(after)
	result.Relationships.Added = stringDifference(afterRelations, beforeRelations)
	result.Relationships.Removed = stringDifference(beforeRelations, afterRelations)
	for _, section := range []*diffSection{&result.Contexts, &result.Modules, &result.Relationships} {
		sortStrings(section.Added)
		sortStrings(section.Removed)
		sortStrings(section.Changed)
	}
	return result
}

func mapDifference[T any](before, after map[string]T) []string {
	result := make([]string, 0)
	for name := range before {
		if _, exists := after[name]; !exists {
			result = append(result, name)
		}
	}
	return result
}

func stringDifference(before, after map[string]bool) []string {
	result := make([]string, 0)
	for name := range before {
		if !after[name] {
			result = append(result, name)
		}
	}
	return result
}

func relationNames(architecture *model.Architecture) map[string]bool {
	result := make(map[string]bool, len(architecture.Relations))
	for _, relation := range architecture.Relations {
		result[relation.From+" -> "+relation.To+" via "+relation.Via] = true
	}
	return result
}

func jsonValue(value any) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func sortStrings(values []string) {
	sort.Strings(values)
}

func printDiffSection(title string, values []string) {
	if len(values) == 0 {
		return
	}
	fmt.Println(title + ":")
	for _, value := range values {
		fmt.Println("  - " + value)
	}
}

func runDoctor(arguments []string) {
	arguments = moveFlagsBeforePath(arguments, "--json")
	flags := flag.NewFlagSet("doctor", flag.ExitOnError)
	jsonOutput := flags.Bool("json", false, "emit machine-readable results")
	flags.Usage = func() { fail("usage: bound doctor [--json] [architecture.bo|directory]") }
	flags.Parse(arguments)
	if len(flags.Args()) > 1 {
		fail("usage: bound doctor [--json] [architecture.bo|directory]")
	}
	location := "."
	if len(flags.Args()) == 1 {
		location = flags.Args()[0]
	}
	result := diagnose(location)
	if *jsonOutput {
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			fail("encode doctor results: " + err.Error())
		}
	} else {
		fmt.Printf("Bound doctor (%s)\n", result.Path)
		for _, item := range result.Checks {
			fmt.Printf("%-7s %s: %s\n", item.Status, item.Name, item.Message)
		}
	}
	if !result.OK {
		os.Exit(1)
	}
}

type doctorResult struct {
	OK     bool          `json:"ok"`
	Path   string        `json:"path"`
	Checks []doctorCheck `json:"checks"`
}

type doctorCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func diagnose(location string) doctorResult {
	absolute, err := filepath.Abs(location)
	if err != nil {
		return doctorResult{Path: location, Checks: []doctorCheck{{Name: "path", Status: "error", Message: err.Error()}}}
	}
	result := doctorResult{OK: true, Path: absolute}
	add := func(name, status, message string) {
		result.Checks = append(result.Checks, doctorCheck{Name: name, Status: status, Message: message})
		if status == "error" {
			result.OK = false
		}
	}
	architecturePath := absolute
	if info, statErr := os.Stat(absolute); statErr == nil && info.IsDir() {
		architecturePath = filepath.Join(absolute, "architecture.bo")
	}
	if _, statErr := os.Stat(architecturePath); statErr != nil {
		add("architecture", "error", "no architecture.bo found at "+architecturePath)
	} else {
		add("architecture", "ok", architecturePath)
		program, compileErr := compiler.Compile(architecturePath, compiler.Options{SkipImplementation: true})
		if compileErr != nil {
			add("model", "error", compileErr.Error())
		} else {
			add("model", "ok", "architecture parses and validates")
			if _, rootErr := os.Stat(program.SourceRoot); rootErr != nil {
				add("implementation", "warning", "implementation root is unavailable: "+program.SourceRoot)
			} else {
				add("implementation", "ok", program.SourceRoot)
			}
		}
	}
	if _, lookErr := exec.LookPath("bound-lsp"); lookErr != nil {
		add("lsp", "warning", "bound-lsp is not available on PATH")
	} else {
		add("lsp", "ok", "bound-lsp is available on PATH")
	}
	return result
}

func runReview(arguments []string) {
	arguments = moveFlagsBeforePath(arguments, "--no-open", "--design-only")
	flags := flag.NewFlagSet("review", flag.ExitOnError)
	noOpen := flags.Bool("no-open", false, "write the review without opening a browser")
	designOnly := flags.Bool("design-only", false, "validate the model without inspecting implementation files")
	flags.Usage = func() { fail("usage: bound review [--no-open] [--design-only] <file.bo>") }
	flags.Parse(arguments)
	architecturePath := onePath(flags.Args(), "review")
	program, err := compiler.Compile(architecturePath, compiler.Options{SkipImplementation: *designOnly})
	if err != nil {
		failError(err, false)
	}
	outputPath := filepath.Join(filepath.Dir(architecturePath), strings.TrimSuffix(filepath.Base(architecturePath), filepath.Ext(architecturePath))+".html")
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		fail("create HTML output directory: " + err.Error())
	}
	if err := os.WriteFile(outputPath, []byte(render.MermaidHTML(program.Architecture)), 0o644); err != nil {
		fail("write HTML architecture review: " + err.Error())
	}
	if !*noOpen {
		if err := openReview(outputPath); err != nil {
			fail(fmt.Sprintf("open HTML architecture review: %v (file written to %s)", err, outputPath))
		}
	}
	fmt.Printf("wrote HTML architecture review %s\n", outputPath)
}

func printDiagnostics(err error) {
	var compilationError *compiler.Error
	if errors.As(err, &compilationError) {
		_ = json.NewEncoder(os.Stdout).Encode(struct {
			OK          bool                  `json:"ok"`
			Diagnostics []compiler.Diagnostic `json:"diagnostics"`
		}{Diagnostics: compilationError.Diagnostics})
		return
	}
	_ = json.NewEncoder(os.Stdout).Encode(struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}{Error: err.Error()})
}

func moveFlagsBeforePath(arguments []string, flags ...string) []string {
	known := make(map[string]bool, len(flags))
	for _, flagName := range flags {
		known[flagName] = true
	}
	ordered := make([]string, 0, len(arguments))
	paths := make([]string, 0, len(arguments))
	for _, argument := range arguments {
		if known[argument] {
			ordered = append(ordered, argument)
		} else {
			paths = append(paths, argument)
		}
	}
	return append(ordered, paths...)
}

func onePath(arguments []string, command string) string {
	if len(arguments) != 1 {
		fail("usage: bound " + command + " <file.bo>")
	}
	return arguments[0]
}

func openReview(path string) error {
	var command string
	var arguments []string
	switch runtime.GOOS {
	case "darwin":
		command, arguments = "open", []string{path}
	case "windows":
		command, arguments = "cmd", []string{"/c", "start", "", path}
	default:
		command, arguments = "xdg-open", []string{path}
	}
	return exec.Command(command, arguments...).Start()
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: bound <command> [options]

commands:
  init       scaffold an architecture.bo file
  check      validate an architecture, optionally with --watch or --json
  compile    emit the resolved compiler IR as JSON
  fmt        format a .bo file, or check formatting in CI
  review     generate an HTML architecture review
  doctor     diagnose workspace and editor setup
  diff       compare two architecture specifications
  version    print the Bound version`)
	os.Exit(2)
}

func fail(message string) { fmt.Fprintln(os.Stderr, "bound: "+message); os.Exit(1) }

func failError(err error, jsonOutput bool) {
	if jsonOutput {
		printDiagnostics(err)
	} else {
		fail(err.Error())
	}
}
