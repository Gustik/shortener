package ignored

// Ignored не должна получить Reset() — нет маркера.
type Ignored struct {
	Value string
}
