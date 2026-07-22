package customers

type Customer struct {
	ID    string
	Email string
}

type CustomerPort interface {
	Find(id string) Customer
}

type Service struct{}

func (Service) Find(id string) Customer {
	return Customer{ID: id, Email: id + "@example.com"}
}
