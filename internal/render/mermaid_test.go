package render

import (
	"strings"
	"testing"

	"bound/internal/model"
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

	output := interactionDiagram(architecture)
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
		"classDiagram",
	} {
		if !strings.Contains(output, expected) {
			t.Errorf("HTML output does not contain %q:\n%s", expected, output)
		}
	}
}
