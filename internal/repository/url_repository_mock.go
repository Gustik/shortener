package repository

import "context"

type MockURLRepository struct {
	urls map[string]string
}

func NewMockURLRepository() *MockURLRepository {
	return &MockURLRepository{
		urls: make(map[string]string),
	}
}

func (r *MockURLRepository) Save(ctx context.Context, id, originalURL string) error {
	if _, exists := r.urls[id]; exists {
		return ErrURLExists
	}

	r.urls[id] = originalURL
	return nil
}

func (r *MockURLRepository) GetByID(ctx context.Context, id string) (string, error) {
	originalURL, exists := r.urls[id]
	if !exists {
		return "", ErrURLNotFound
	}

	return originalURL, nil
}
