package web

import "testing"

func TestCheckout(t *testing.T) {
	if got, want := Checkout("customer-1", "order-1", 42), "order order-1 for customer-1@example.com"; got != want {
		t.Fatalf("Checkout() = %q, want %q", got, want)
	}
}
