package customers

type CustomerPort interface {
	Find(id string) Customer
}
