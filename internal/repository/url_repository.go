package repository

import (
	"errors"
	"sync"
)

var (
	ErrURLNotFound = errors.New("URL not found")
	ErrURLExists   = errors.New("URL already exists")
)

type URLRepository interface {
	Save(id, originalURL string) error
	GetByID(id string) (string, error)
}

type InMemoryURLRepository struct {
	mu   sync.RWMutex
	urls map[string]string
}

func NewInMemoryURLRepository() *InMemoryURLRepository {
	return &InMemoryURLRepository{
		urls: make(map[string]string),
	}
}

func (r *InMemoryURLRepository) Save(id, originalURL string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.urls[id]; exists {
		return ErrURLExists
	}

	r.urls[id] = originalURL
	return nil
}

func (r *InMemoryURLRepository) GetByID(id string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	originalURL, exists := r.urls[id]
	if !exists {
		return "", ErrURLNotFound
	}

	return originalURL, nil
}
