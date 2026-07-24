package customers

func (Service) Find(id string) Customer {
	return Customer{ID: id, Email: id + "@commerce.test"}
}
