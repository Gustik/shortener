package repository

import (
	"context"
	"slices"
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

func (r *InMemoryURLRepository) Save(ctx context.Context, shortURL, originalURL, userID string) (*model.URLRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.urls {
		if r.urls[i].ShortURL == shortURL {
			return nil, ErrShortURLConflict
		}
		if r.urls[i].OriginalURL == originalURL {
			return &r.urls[i], ErrURLConflict
		}
	}

	record := model.URLRecord{UUID: uuid.New(), ShortURL: shortURL, OriginalURL: originalURL, UserID: userID}
	r.urls = append(r.urls, record)

	return &record, nil
}

func (r *InMemoryURLRepository) GetByShortURL(ctx context.Context, shortURL string) (*model.URLRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.urls {
		if r.urls[i].ShortURL == shortURL {
			if r.urls[i].IsDeleted {
				return nil, ErrURLDeleted
			}
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
			if r.urls[j].ShortURL == record.ShortURL {
				return nil, ErrShortURLConflict
			}
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

func (r *InMemoryURLRepository) GetByUserID(ctx context.Context, userID string) ([]model.URLRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var records []model.URLRecord
	for i := range r.urls {
		if r.urls[i].UserID == userID {
			records = append(records, r.urls[i])
		}
	}

	return records, nil
}

func (r *InMemoryURLRepository) DeleteURLs(ctx context.Context, shortURLs []string, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range r.urls {
		if r.urls[i].UserID != userID {
			continue
		}
		if slices.Contains(shortURLs, r.urls[i].ShortURL) {
			r.urls[i].IsDeleted = true
		}
	}

	return nil
}

func (r *InMemoryURLRepository) Ping(ctx context.Context) error {
	return nil
}
