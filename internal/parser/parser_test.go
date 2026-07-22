package parser

import (
	"strings"
	"testing"
)

func TestParseBoundDocument(t *testing.T) {
	a, err := Parse(strings.NewReader(`architecture Commerce {
  context Orders {
    implementation go "github.com/acme/commerce/internal/orders"
  }
  context Customers {
    implementation rust "crates/customers"
    exposes CustomerPort
  }
  Orders -> Customers via CustomerPort
}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Validate(); err != nil {
		t.Fatal(err)
	}
	if a.Contexts["Orders"].Implementation.Language != "go" {
		t.Fatal("expected Go implementation")
	}
	if a.Contexts["Customers"].Implementation.Language != "rust" {
		t.Fatal("expected Rust implementation")
	}
}
