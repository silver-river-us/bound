package orders

type OrderPort interface {
	Place(id string, amount int) Order
}
