package pointers

// generate:reset
// Session представляет сессию с указателями на примитивы и вложенной структурой.
type Session struct {
	Token    string
	TokenPtr *string
	Count    *int
	Child    *Session
}
