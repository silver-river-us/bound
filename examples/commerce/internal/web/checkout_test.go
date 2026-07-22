package web

import "testing"

import (
	"example.com/commerce/internal/customers"
	"example.com/commerce/internal/orders"
)

func TestCheckout(t *testing.T) {
	if got, want := Checkout(customers.Service{}, orders.Service{}, "customer-1", "order-1", 42), "order order-1 for customer-1@example.com"; got != want {
		t.Fatalf("Checkout() = %q, want %q", got, want)
	}
}
