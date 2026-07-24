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
  object Customer do
    attribute :id :string
    attribute :email :string
  end
  context Orders do
    implementation go "./internal/orders"
    interface OrderPort do
      behavior Place(orderID string, amount int) Order
    end
    exposes OrderPort
  end
	  context Customers do
    implementation rust "./crates/customers"
    interface CustomerPort do
      behavior Find(customerID string) Customer
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
	if a.Objects["Customer"].Attributes["email"].Type != "string" {
		t.Fatal("expected typed Customer.email attribute")
	}
	if a.Contexts["Customers"].Implementation.Language != "rust" {
		t.Fatal("expected Rust implementation")
	}
}

func TestParseMap(t *testing.T) {
	name, files, err := ParseMap(strings.NewReader(`map Commerce do
  "internal/orders/order.go" -> Orders
  entrypoint "cmd/commerce/main.go" -> App
end`))
	if err != nil {
		t.Fatal(err)
	}
	if name != "Commerce" || len(files) != 2 || !files[1].EntryPoint {
		t.Fatalf("unexpected map: %s %#v", name, files)
	}
}

func TestParseRejectsNonSymbolAttributeSyntax(t *testing.T) {
	_, err := Parse(strings.NewReader(`architecture Example do
  object Customer do
    attribute email: string
  end
end`))
	if err == nil {
		t.Fatal("expected legacy attribute syntax to be rejected")
	}
}
