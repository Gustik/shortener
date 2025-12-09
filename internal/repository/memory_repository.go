package repository

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"github.com/Gustik/shortener/internal/model"
)

type InMemoryURLRepository struct {
	mu   sync.Mutex
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
			return &r.urls[i], fmt.Errorf("%s - %w", originalURL, ErrURLExists)
		}
	}

	record := model.URLRecord{UUID: uuid.New(), ShortURL: shortURL, OriginalURL: originalURL}
	r.urls = append(r.urls, record)

	return &record, nil
}

func (r *InMemoryURLRepository) GetByShortURL(ctx context.Context, shortURL string) (*model.URLRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.urls {
		if r.urls[i].ShortURL == shortURL {
			return &r.urls[i], nil
		}
	}

	return nil, ErrURLNotFound
}

func (r *InMemoryURLRepository) SaveBatch(ctx context.Context, records []model.URLRecord) ([]model.URLRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := make([]model.URLRecord, len(records))

	for i, record := range records {
		// Проверяем, существует ли уже такой original_url
		existingRecord := (*model.URLRecord)(nil)
		for j := range r.urls {
			if r.urls[j].OriginalURL == record.OriginalURL {
				existingRecord = &r.urls[j]
				break
			}
		}

		if existingRecord != nil {
			result[i] = *existingRecord
		} else {
			record.UUID = uuid.New()
			r.urls = append(r.urls, record)
			result[i] = record
		}
	}

	return result, nil
}

func (r *InMemoryURLRepository) Ping(ctx context.Context) error {
	return nil
}
