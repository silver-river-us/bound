package model_test

import (
	. "bound/src/lib/model"
	"strings"
	"testing"
)

func validArchitectureForQuality() *Architecture {
	return &Architecture{
		Name:           "Example",
		Implementation: Implementation{Language: "go", Locator: "."},
		Contexts:       map[string]*Context{},
		Objects:        map[string]*Object{},
		Modules:        map[string]*Module{},
	}
}

func contract(name string) *Interface {
	return &Interface{Name: name, Types: map[string]*Object{}, Operations: map[string]Operation{}}
}

func context(name string) *Context {
	return &Context{Name: name, Exposes: map[string]bool{}, ExposeSpans: map[string]Span{}, Interfaces: map[string]*Interface{}, Modules: map[string]*Module{}}
}

func TestValidationAllowsReusableArchitectureDomainTypesInContracts(t *testing.T) {
	architecture := validArchitectureForQuality()
	architecture.Objects["Customer"] = &Object{
		Name: "Customer", Kind: "entity",
		Attributes: map[string]Attribute{"id": {Name: "id", Type: "string"}},
		Operations: map[string]Operation{},
	}
	orders := context("Orders")
	orders.Exposes["OrdersAPI"] = true
	orders.Interfaces["OrdersAPI"] = &Interface{
		Name:  "OrdersAPI",
		Types: map[string]*Object{},
		Operations: map[string]Operation{
			"find": {Name: "find", Parameters: []Parameter{{Name: "customer", Type: "Customer"}}, Returns: "Customer"},
		},
	}
	architecture.Contexts[orders.Name] = orders

	if err := architecture.Validate(); err != nil {
		t.Fatalf("validate architecture: %v", err)
	}
}

func TestValidationRejectsDuplicateRelationships(t *testing.T) {
	architecture := validArchitectureForQuality()
	architecture.Contexts["Orders"] = context("Orders")
	architecture.Contexts["Billing"] = context("Billing")
	architecture.Relations = []Relation{{From: "Orders", To: "Billing"}, {From: "Orders", To: "Billing"}}

	if err := architecture.Validate(); err == nil || !strings.Contains(err.Error(), "duplicate relationship") {
		t.Fatalf("error = %v, want duplicate relationship error", err)
	}
}

func TestValidationRejectsUnreachableContext(t *testing.T) {
	architecture := validArchitectureForQuality()
	architecture.Contexts["Orders"] = context("Orders")
	architecture.Contexts["Billing"] = context("Billing")

	if err := architecture.Validate(); err == nil || !strings.Contains(err.Error(), "is unreachable") {
		t.Fatalf("error = %v, want unreachable context error", err)
	}
}

func TestValidationRejectsUnusedContract(t *testing.T) {
	architecture := validArchitectureForQuality()
	orders := context("Orders")
	orders.Interfaces["Internal"] = contract("Internal")
	architecture.Contexts[orders.Name] = orders

	if err := architecture.Validate(); err == nil || !strings.Contains(err.Error(), "interface Orders.Internal is unused") {
		t.Fatalf("error = %v, want unused contract error", err)
	}
}

func TestValidationRejectsSuspiciousContextCycle(t *testing.T) {
	architecture := validArchitectureForQuality()
	architecture.Contexts["Orders"] = context("Orders")
	architecture.Contexts["Billing"] = context("Billing")
	architecture.Relations = []Relation{{From: "Orders", To: "Billing"}, {From: "Billing", To: "Orders"}}

	if err := architecture.Validate(); err == nil || !strings.Contains(err.Error(), "suspicious dependency cycle") {
		t.Fatalf("error = %v, want suspicious dependency cycle error", err)
	}
}
