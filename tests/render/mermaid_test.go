package render_test

import (
	. "bound/src/boundary/render"
	"encoding/xml"
	"strings"
	"testing"

	"bound/src/lib/model"
)

func TestMermaidMarkdownRendersArchitectureDiagrams(t *testing.T) {
	architecture := &model.Architecture{
		Name:        "Example",
		Description: "A small example",
		Implementation: model.Implementation{
			Language: "go",
			Locator:  "./",
		},
		Contexts: map[string]*model.Context{
			"Reporting": {
				Name: "Reporting",
				Interfaces: map[string]*model.Interface{
					"Reports": {Name: "Reports"},
				},
				Modules: map[string]*model.Module{
					"Reporting.Daily": {
						Name:       "Daily",
						Qualified:  "Reporting.Daily",
						Context:    "Reporting",
						Implements: "Reports",
						Uses:       map[string]bool{"Reporting.Helper": true},
					},
					"Reporting.Helper": {
						Name:      "Helper",
						Qualified: "Reporting.Helper",
						Context:   "Reporting",
					},
				},
			},
			"Source": {
				Name:       "Source",
				Interfaces: map[string]*model.Interface{},
				Modules:    map[string]*model.Module{},
			},
		},
		Relations: []model.Relation{{From: "Reporting", To: "Source", Via: "Reports"}},
		Files: []model.FileMapping{
			{Path: "main.go", Node: "", RootEntrypoint: true, EntryPoint: true, EntryPointName: "main"},
			{Path: "reporting/daily.go", Node: "Reporting.Daily"},
		},
	}

	output := MermaidMarkdown(architecture)
	for _, expected := range []string{
		"# Architecture: Example",
		"## Context relationships",
		"## Component view",
		"## Interaction view (declared relationships)",
		"## Contracts and modules",
		"## Domain model",
		"## Source ownership",
		"classDiagram",
		"subgraph component_context_Reporting[\"Reporting\"]",
		"participant participant_Reporting as Reporting",
		"participant participant_Source as Source",
		"participant_Reporting->>participant_Source: via Reports",
		"<<context>> context_Reporting",
		"context_Reporting --> context_Source : via Reports",
		"module_Reporting_Daily ..|> interface_Reporting_Reports : implements",
		"module_Reporting_Daily ..> module_Reporting_Helper : uses",
		"subgraph source_module_Reporting_Daily[\"Reporting.Daily\"]",
		"subgraph source_module_root[\"Example (root)\"]",
		"file_reporting_daily_go[\"daily.go\"]",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("Mermaid output does not contain %q:\n%s", expected, output)
		}
	}
}

func TestDomainDiagramRendersObjectRelationships(t *testing.T) {
	architecture := &model.Architecture{
		Objects: map[string]*model.Object{
			"Application": {
				Name: "Application",
				Kind: "entity",
				Attributes: map[string]model.Attribute{
					"product": {Name: "product", Type: "Product"},
				},
			},
			"Product": {Name: "Product", Kind: "entity", Attributes: map[string]model.Attribute{}},
		},
	}

	output := DomainDiagram(architecture)
	for _, expected := range []string{
		"classDiagram",
		`class object_Application["Application"]`,
		"object_Application : +product Product",
		"object_Application --> object_Product : product",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("domain output does not contain %q:\n%s", expected, output)
		}
	}
}

func TestInteractionDiagramUsesModuleDependenciesWhenContextsHaveNoRelations(t *testing.T) {
	architecture := &model.Architecture{
		Contexts: map[string]*model.Context{
			"Reporting": {
				Name: "Reporting",
				Interfaces: map[string]*model.Interface{
					"Reports": {Name: "Reports"},
				},
			},
		},
		Modules: map[string]*model.Module{
			"Reporting.Daily": {
				Qualified: "Reporting.Daily",
				Context:   "Reporting",
				Uses:      map[string]bool{"Reports": true},
			},
		},
	}

	output := InteractionDiagram(architecture)
	for _, expected := range []string{
		"participant module_Reporting_Daily as Reporting.Daily",
		"participant interface_Reporting_Reports as Reports",
		"module_Reporting_Daily->>interface_Reporting_Reports: uses Reports",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("interaction output does not contain %q:\n%s", expected, output)
		}
	}
}

func TestDomainHTMLScopesIncludeContextTypes(t *testing.T) {
	architecture := &model.Architecture{
		Contexts: map[string]*model.Context{
			"Reporting": {
				Name: "Reporting",
				Interfaces: map[string]*model.Interface{
					"Reports": {Name: "Reports", Types: map[string]*model.Object{
						"Snapshot": {Name: "Snapshot", Kind: "value", Attributes: map[string]model.Attribute{}},
					}},
				},
			},
		},
	}

	output := MermaidHTML(architecture)
	for _, expected := range []string{
		`data-domain-scope="domain_Reporting"`,
		`class object_Reports_Snapshot[&#34;Reports.Snapshot&#34;]`,
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("scoped HTML output does not contain %q:\n%s", expected, output)
		}
	}
}

func TestMermaidHTMLRendersInteractiveReviewPage(t *testing.T) {
	architecture := &model.Architecture{
		Name:           "Example",
		Implementation: model.Implementation{Language: "go", Locator: "."},
		Contexts: map[string]*model.Context{
			"Source": {Name: "Source", Interfaces: map[string]*model.Interface{}, Modules: map[string]*model.Module{}},
		},
		Modules: map[string]*model.Module{},
	}

	output := MermaidHTML(architecture)
	for _, expected := range []string{
		"<!doctype html>",
		"Architecture: Example",
		"https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs",
		"Context relationships",
		"data-action=\"fullscreen\"",
		"aria-label=\"Architecture summary\"",
		"href=\"#domain-model\"",
		"data-domain-filter",
		"All architecture types",
		"classDiagram",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("HTML output does not contain %q:\n%s", expected, output)
		}
	}
}

func TestMermaidHTMLOptionsSupportOfflineAssetsAndPrinting(t *testing.T) {
	architecture := &model.Architecture{Name: "Offline", Contexts: map[string]*model.Context{}}
	local := MermaidHTMLWithOptions(architecture, HTMLOptions{MermaidURL: "./assets/mermaid.mjs", PrintFriendly: true})
	for _, expected := range []string{`import mermaid from "./assets/mermaid.mjs"`, `media="print"`, `nav, .controls`} {
		if !strings.Contains(local, expected) {
			t.Errorf("local HTML does not contain %q", expected)
		}
	}
	if strings.Contains(local, DefaultMermaidURL) {
		t.Error("local HTML unexpectedly includes the CDN asset")
	}

	inline := MermaidHTMLWithOptions(architecture, HTMLOptions{InlineMermaid: "globalThis.mermaid = {};"})
	if !strings.Contains(inline, "globalThis.mermaid = {};") || strings.Contains(inline, "import mermaid from") {
		t.Error("inline HTML did not use the embedded asset contract")
	}
}

func TestExportFormatsAndSVGAreDeterministicAndValid(t *testing.T) {
	architecture := &model.Architecture{Name: "Export", Contexts: map[string]*model.Context{}}
	markdown, err := Export(architecture, ExportOptions{Format: FormatMarkdown})
	if err != nil || !strings.Contains(markdown, "# Architecture: Export") {
		t.Fatalf("markdown export = %q, err = %v", markdown, err)
	}
	svg, err := Export(architecture, ExportOptions{Format: FormatSVG})
	if err != nil {
		t.Fatalf("SVG export: %v", err)
	}
	var document struct{}
	if err := xml.Unmarshal([]byte(svg), &document); err != nil {
		t.Fatalf("SVG is not XML: %v", err)
	}
	if !strings.Contains(svg, `<svg xmlns="http://www.w3.org/2000/svg"`) || !strings.Contains(svg, "Context relationships") {
		t.Errorf("SVG export is missing required review content: %s", svg)
	}
	if _, err := Export(architecture, ExportOptions{Format: "pdf"}); err == nil {
		t.Fatal("unsupported format unexpectedly succeeded")
	}
}

func TestHTMLBrowserSmokeContractWithoutBrowser(t *testing.T) {
	architecture := &model.Architecture{Name: "Smoke", Contexts: map[string]*model.Context{}}
	output := MermaidHTML(architecture)
	checks := []string{
		`<script type="module">`,
		`mermaid.initialize({ startOnLoad: false`,
		`await mermaid.render(`,
		`catch (error)`,
		`class=\"error\"`,
		`document.addEventListener("click"`,
		`data-action="in"`,
		`data-action="out"`,
		`data-action="reset"`,
		`data-action="fullscreen"`,
	}
	for _, check := range checks {
		if !strings.Contains(output, check) {
			t.Errorf("generated HTML is missing browser smoke contract %q", check)
		}
	}
	if strings.Count(output, "<script") != strings.Count(output, "</script>") {
		t.Error("generated HTML has unbalanced script tags")
	}
}
