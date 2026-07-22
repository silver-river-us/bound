package main

import (
	"fmt"
	"example.com/commerce/internal/web"
)

func main() {
	fmt.Println(web.Checkout("customer-1", "order-1", 42))
}
