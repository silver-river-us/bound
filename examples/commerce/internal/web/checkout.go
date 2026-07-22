package web

import (
	"fmt"

	"example.com/commerce/internal/customers"
	"example.com/commerce/internal/orders"
)

func Checkout(customerPort customers.CustomerPort, orderPort orders.OrderPort, customerID, orderID string, amount int) string {
	customer := customerPort.Find(customerID)
	order := orderPort.Place(orderID, amount)
	return fmt.Sprintf("order %s for %s", order.ID, customer.Email)
}
