package customers

type Customer struct {
	ID   string
	Email string
}

func Find(id string) Customer {
	return Customer{ID: id, Email: id + "@example.com"}
}
