package web

import (
	"fmt"

	"commerce/internal/customers"
	"commerce/internal/orders"
)

func Checkout(customerPort customers.CustomerPort, orderPort orders.OrderPort, customerID, orderID string, amount int) string {
	customer := customerPort.Find(customerID)
	order := orderPort.Place(orderID, amount)
	return fmt.Sprintf("order %s for %s", order.ID, customer.Email)
}
