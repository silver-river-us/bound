package model_test

import (
	. "bound/src/model"
	"testing"
)

func TestModuleAllowsDeclaredInterfaceAndPrivateModuleDependencies(t *testing.T) {
	architecture := &Architecture{
		Contexts: map[string]*Context{
			"Source": {
				Name: "Source",
				Exposes: map[string]bool{
					"Public": true,
				},
				Interfaces: map[string]*Interface{
					"Public": {Name: "Public"},
				},
			},
			"Consumer": {
				Name:       "Consumer",
				Exposes:    map[string]bool{},
				Interfaces: map[string]*Interface{},
			},
		},
		Modules: map[string]*Module{},
		Relations: []Relation{
			{From: "Consumer", To: "Source", Via: "Public"},
		},
	}
	provider := &Module{Name: "Provider", Qualified: "Source.Provider", Context: "Source", Implements: "Public"}
	helper := &Module{Name: "Helper", Qualified: "Consumer.Helper", Context: "Consumer"}
	consumer := &Module{
		Name:      "App",
		Qualified: "Consumer.App",
		Context:   "Consumer",
		Uses: map[string]bool{
			"Source.Public":   true,
			"Consumer.Helper": true,
		},
	}
	architecture.Modules[provider.Qualified] = provider
	architecture.Modules[helper.Qualified] = helper
	architecture.Modules[consumer.Qualified] = consumer

	if !architecture.ModuleAllows(consumer.Qualified, provider.Qualified) {
		t.Fatal("declared exposed interface dependency was rejected")
	}
	if !architecture.ModuleAllows(consumer.Qualified, helper.Qualified) {
		t.Fatal("declared private module dependency was rejected")
	}
	if architecture.ModuleAllows(helper.Qualified, provider.Qualified) {
		t.Fatal("undeclared module dependency was allowed")
	}
}

func TestValidationRejectsSourceMappingWithoutPrivateModule(t *testing.T) {
	architecture := &Architecture{
		Name:           "Example",
		Implementation: Implementation{Language: "go", Locator: "./"},
		Contexts: map[string]*Context{
			"Reporting": {Name: "Reporting", Exposes: map[string]bool{}, Interfaces: map[string]*Interface{}},
		},
		Modules: map[string]*Module{},
		Objects: map[string]*Object{},
		Files:   []FileMapping{{Path: "report.go", Node: "Reporting"}},
	}

	if err := architecture.Validate(); err == nil {
		t.Fatal("context source mapping was accepted")
	}
}
