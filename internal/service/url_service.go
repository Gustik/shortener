package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"go.uber.org/zap"

	"github.com/Gustik/shortener/internal/repository"
)

var (
	ErrEmptyURL     = errors.New("URL cannot be empty")
	ErrEmptyShortID = errors.New("ShortID cannot be empty")
	ErrURLNotFound  = errors.New("URL not found")
)

type URLService interface {
	ShortenURL(ctx context.Context, originalURL string) (string, error)
	GetOriginalURL(ctx context.Context, shortID string) (string, error)
	Ping(ctx context.Context) error
}

type urlService struct {
	repo    repository.URLRepository
	baseURL string
	logger  *zap.Logger
}

func NewURLService(repo repository.URLRepository, baseURL string, logger *zap.Logger) URLService {
	return &urlService{
		repo:    repo,
		baseURL: baseURL,
		logger:  logger,
	}
}

func (s *urlService) ShortenURL(ctx context.Context, originalURL string) (string, error) {
	if originalURL == "" {
		return "", ErrEmptyURL
	}

	shortURL := s.generateShortURL()

	savedURL, err := s.repo.Save(ctx, shortURL, originalURL)
	if errors.Is(err, repository.ErrURLExists) {
		s.logger.Sugar().Infof("%s", err.Error())
		return fmt.Sprintf("%s/%s", s.baseURL, savedURL.ShortURL), nil
	}

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s", s.baseURL, savedURL.ShortURL), nil
}

func (s *urlService) GetOriginalURL(ctx context.Context, shortID string) (string, error) {
	if shortID == "" {
		return "", ErrEmptyShortID
	}

	url, err := s.repo.GetByShortURL(ctx, shortID)
	if errors.Is(err, repository.ErrURLNotFound) {
		return "", ErrURLNotFound
	}

	return url.OriginalURL, nil
}

func (s *urlService) Ping(ctx context.Context) error {
	return s.repo.Ping(ctx)
}

func (s *urlService) generateShortURL() string {
	b := make([]byte, 6)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:8]
}
