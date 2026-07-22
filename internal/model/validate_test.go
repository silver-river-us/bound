package model

import "testing"

func TestValidateRequiresExposedContract(t *testing.T) {
	a := &Architecture{
		Name: "Commerce",
		Contexts: map[string]*Context{
			"Orders":    {Name: "Orders", Implementation: Implementation{Language: "go", Locator: "internal/orders"}},
			"Customers": {Name: "Customers", Implementation: Implementation{Language: "go", Locator: "internal/customers"}, Exposes: map[string]bool{"CustomerPort": true}},
		},
		Relations: []Relation{{From: "Orders", To: "Customers", Via: "CustomerPort"}},
	}
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsUnknownContext(t *testing.T) {
	a := &Architecture{Name: "Commerce", Contexts: map[string]*Context{"Orders": {Name: "Orders", Implementation: Implementation{Language: "go", Locator: "internal/orders"}}}, Relations: []Relation{{From: "Orders", To: "Payments"}}}
	if err := a.Validate(); err == nil {
		t.Fatal("expected unknown context error")
	}
}
