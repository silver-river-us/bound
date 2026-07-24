package orders

func (Service) Place(id string, amount int) Order {
	return Order{ID: id, Amount: amount}
}
