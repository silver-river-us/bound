package web

import (
	"fmt"

	"example.com/commerce/internal/customers"
	"example.com/commerce/internal/orders"
)

func Checkout(customerID, orderID string, amount int) string {
	customer := customers.Find(customerID)
	order := orders.Place(orderID, amount)
	return fmt.Sprintf("order %s for %s", order.ID, customer.Email)
}
