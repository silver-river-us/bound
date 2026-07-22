package orders

type Order struct {
	ID     string
	Amount int
}

type OrderPort interface {
	Place(id string, amount int) Order
}

type Service struct{}

func (Service) Place(id string, amount int) Order {
	return Order{ID: id, Amount: amount}
}
