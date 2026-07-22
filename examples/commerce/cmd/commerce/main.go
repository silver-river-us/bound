package main

import (
	"example.com/commerce/internal/customers"
	"example.com/commerce/internal/orders"
	"example.com/commerce/internal/web"
	"fmt"
)

func main() {
	fmt.Println(web.Checkout(customers.Service{}, orders.Service{}, "customer-1", "order-1", 42))
}
