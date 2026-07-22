package parser

import (
	"strings"
	"testing"
)

func TestParseBoundDocument(t *testing.T) {
	a, err := Parse(strings.NewReader(`"""
Architecture documentation.
"""
architecture Commerce do
  context Orders do
    implementation go "./internal/orders"
    interface OrderPort do
      operation Place(orderID string, amount int) Order
    end
    exposes OrderPort
  end
  context Customers do
    implementation rust "./crates/customers"
    interface CustomerPort do
      operation Find(customerID string) Customer
    end
    exposes CustomerPort
  end
  Orders -> Customers via CustomerPort
end`))
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
	if a.Contexts["Orders"].Implementation.Language != "go" {
		t.Fatal("expected Go implementation")
	}
	if _, ok := a.Contexts["Orders"].Interfaces["OrderPort"]; !ok {
		t.Fatal("expected OrderPort interface")
	}
	if a.Description != "Architecture documentation." {
		t.Fatalf("unexpected architecture description: %q", a.Description)
	}
	if a.Contexts["Customers"].Implementation.Language != "rust" {
		t.Fatal("expected Rust implementation")
	}
}
