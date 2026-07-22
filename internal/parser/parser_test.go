package parser

import (
	"strings"
	"testing"
)

func TestParseBoundDocument(t *testing.T) {
	a, err := Parse(strings.NewReader(`architecture Commerce do
  context Orders do
    implementation go "github.com/acme/commerce/internal/orders"
  end
  context Customers do
    implementation rust "crates/customers"
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
	if a.Contexts["Customers"].Implementation.Language != "rust" {
		t.Fatal("expected Rust implementation")
	}
}
