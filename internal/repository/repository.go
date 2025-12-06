package repository

import (
	"context"
	"errors"

	"github.com/Gustik/shortener/internal/model"
)

var (
	ErrURLNotFound = errors.New("URL not found")
	ErrURLExists   = errors.New("URL already exists")
)

type URLRepository interface {
	Save(ctx context.Context, shortURL, originalURL string) (*model.URLRecord, error)
	GetByShortURL(ctx context.Context, shortURL string) (*model.URLRecord, error)
	Ping(ctx context.Context) error
}
