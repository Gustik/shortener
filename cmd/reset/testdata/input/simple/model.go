package simple

// generate:reset
// User представляет пользователя системы.
type User struct {
	ID     int
	Name   string
	Score  float64
	Active bool
	Tags   []string
	Meta   map[string]string
}
