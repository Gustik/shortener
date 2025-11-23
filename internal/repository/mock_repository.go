package repository

import (
	"context"

	"github.com/Gustik/shortener/internal/model"
	"github.com/google/uuid"
)

type MockURLRepository struct {
	urls []model.URLRecord
}

func NewMockURLRepository() *MockURLRepository {
	return &MockURLRepository{
		urls: make([]model.URLRecord, 0),
	}
}

func (r *MockURLRepository) Save(ctx context.Context, shortURL, originalURL string) (*model.URLRecord, error) {
	record := model.URLRecord{
		UUID:        uuid.New(),
		ShortURL:    shortURL,
		OriginalURL: originalURL,
	}
	r.urls = append(r.urls, record)
	return &record, nil
}

func (r *MockURLRepository) GetByShortURL(ctx context.Context, shortURL string) (*model.URLRecord, error) {
	for i := range r.urls {
		if r.urls[i].ShortURL == shortURL {
			return &r.urls[i], nil
		}
	}

	return nil, ErrURLNotFound
}
