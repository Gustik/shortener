package repository

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/Gustik/shortener/internal/model"
)

type InMemoryURLRepository struct {
	mu   sync.RWMutex
	urls []model.URLRecord
}

func NewInMemoryURLRepository() *InMemoryURLRepository {
	return &InMemoryURLRepository{
		urls: make([]model.URLRecord, 0, 10),
	}
}

func (r *InMemoryURLRepository) Save(ctx context.Context, shortURL, originalURL string) (*model.URLRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.urls {
		if r.urls[i].OriginalURL == originalURL {
			return &r.urls[i], ErrURLExists
		}
	}

	r.urls = append(r.urls, model.URLRecord{UUID: uuid.New(), ShortURL: shortURL, OriginalURL: originalURL})

	return nil, nil
}

func (r *InMemoryURLRepository) GetByShortURL(ctx context.Context, shortURL string) (*model.URLRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for i := range r.urls {
		if r.urls[i].ShortURL == shortURL {
			return &r.urls[i], nil
		}
	}

	return nil, ErrURLNotFound
}
