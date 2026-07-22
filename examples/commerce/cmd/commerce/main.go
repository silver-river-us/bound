package main

import (
	"commerce/internal/customers"
	"commerce/internal/orders"
	"commerce/internal/web"
	"fmt"
)

func main() {
	fmt.Println(web.Checkout(customers.Service{}, orders.Service{}, "customer-1", "order-1", 42))
}
