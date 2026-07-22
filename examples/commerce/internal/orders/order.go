package orders

type Order struct {
	ID    string
	Amount int
}

func Place(id string, amount int) Order {
	return Order{ID: id, Amount: amount}
}
