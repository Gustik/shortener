package repository

import (
	"context"
	"errors"

	"github.com/Gustik/shortener/internal/model"
)

// Ошибки уровня репозитория, возвращаемые реализациями URLRepository.
var (
	ErrURLNotFound      = errors.New("URL not found")
	ErrURLConflict      = errors.New("URL already exists")
	ErrShortURLConflict = errors.New("short URL already exists")
	ErrURLDeleted       = errors.New("URL has been deleted")
)

// URLRepository определяет контракт хранения записей URL.
type URLRepository interface {
	// Save сохраняет новую связку короткого и оригинального URL и возвращает запись.
	Save(ctx context.Context, shortURL, originalURL, userID string) (*model.URLRecord, error)
	// SaveBatch сохраняет несколько записей URL за одну операцию.
	SaveBatch(ctx context.Context, records []model.URLRecord) ([]model.URLRecord, error)
	// GetByShortURL возвращает запись URL по короткому идентификатору.
	GetByShortURL(ctx context.Context, shortURL string) (*model.URLRecord, error)
	// GetByUserID возвращает все записи URL, принадлежащие пользователю.
	GetByUserID(ctx context.Context, userID string) ([]model.URLRecord, error)
	// DeleteURLs помечает указанные короткие URL как удалённые для данного пользователя.
	DeleteURLs(ctx context.Context, shortURLs []string, userID string) error
	// Ping проверяет доступность хранилища.
	Ping(ctx context.Context) error
}
