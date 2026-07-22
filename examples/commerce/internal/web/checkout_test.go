package web

import "testing"

import (
	"commerce/internal/customers"
	"commerce/internal/orders"
)

func TestCheckout(t *testing.T) {
	if got, want := Checkout(customers.Service{}, orders.Service{}, "customer-1", "order-1", 42), "order order-1 for customer-1@commerce.test"; got != want {
		t.Fatalf("Checkout() = %q, want %q", got, want)
	}
}
