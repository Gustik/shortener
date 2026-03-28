package pool

import "sync"

// Resettable ограничивает generic-параметр типами, у которых есть метод Reset().
type Resettable interface {
	Reset()
}

// Pool — типобезопасная обёртка над sync.Pool для объектов с методом Reset().
type Pool[T Resettable] struct {
	p sync.Pool
}

// New создаёт Pool, используя fn для создания новых объектов.
func New[T Resettable](fn func() T) *Pool[T] {
	return &Pool[T]{
		p: sync.Pool{
			New: func() any { return fn() },
		},
	}
}

// Get возвращает объект из пула.
func (p *Pool[T]) Get() T {
	return p.p.Get().(T)
}

// Put сбрасывает состояние объекта и возвращает его в пул.
func (p *Pool[T]) Put(v T) {
	v.Reset()
	p.p.Put(v)
}
