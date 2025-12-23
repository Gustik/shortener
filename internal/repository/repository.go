package repository

import (
	"context"
	"errors"

	"github.com/Gustik/shortener/internal/model"
)

var (
	ErrURLNotFound      = errors.New("URL not found")
	ErrURLConflict      = errors.New("URL already exists")
	ErrShortURLConflict = errors.New("short URL already exists")
)

type URLRepository interface {
	Save(ctx context.Context, shortURL, originalURL, userID string) (*model.URLRecord, error)
	SaveBatch(ctx context.Context, records []model.URLRecord) ([]model.URLRecord, error)
	GetByShortURL(ctx context.Context, shortURL string) (*model.URLRecord, error)
	GetByUserID(ctx context.Context, userID string) ([]model.URLRecord, error)
	Ping(ctx context.Context) error
}
