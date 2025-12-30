package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"slices"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/Gustik/shortener/internal/model"
	"github.com/Gustik/shortener/internal/repository"
)

const (
	maxSaveRetries       = 5
	deleteBatchSize      = 10
	deleteMaxConcurrency = 3
)

var (
	ErrEmptyURL           = errors.New("URL cannot be empty")
	ErrEmptyURLBatch      = errors.New("URL batch cannot be empty")
	ErrEmptyShortID       = errors.New("ShortID cannot be empty")
	ErrURLNotFound        = errors.New("URL not found")
	ErrURLExists          = errors.New("URL already exists")
	ErrURLDeleted         = errors.New("URL has been deleted")
	ErrMaxRetriesExceeded = errors.New("maximum retry attempts exceeded for generating unique short URL")
)

type URLService interface {
	ShortenURL(ctx context.Context, originalURL, userID string) (string, error)
	ShortenURLBatch(ctx context.Context, urls []model.BatchRequest, userID string) ([]model.BatchResponse, error)
	GetOriginalURL(ctx context.Context, shortID string) (string, error)
	GetUserURLs(ctx context.Context, userID string) ([]model.UserURLResponse, error)
	DeleteURLs(ctx context.Context, userID string, shortURLs []string)
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

func (s *urlService) ShortenURL(ctx context.Context, originalURL, userID string) (string, error) {
	if originalURL == "" {
		return "", ErrEmptyURL
	}

	for range maxSaveRetries {
		shortURL := s.generateShortURL()

		savedURL, err := s.repo.Save(ctx, shortURL, originalURL, userID)
		if errors.Is(err, repository.ErrURLConflict) {
			s.logger.Sugar().Infof("%s", err.Error())
			return fmt.Sprintf("%s/%s", s.baseURL, savedURL.ShortURL), ErrURLExists
		}

		if errors.Is(err, repository.ErrShortURLConflict) {
			s.logger.Sugar().Infof("%s", err.Error())
			continue
		}

		if err != nil {
			return "", err
		}

		return fmt.Sprintf("%s/%s", s.baseURL, savedURL.ShortURL), nil
	}

	s.logger.Sugar().Errorf("не удалось сгенерировать уникальный short_url после %d попыток", maxSaveRetries)

	return "", ErrMaxRetriesExceeded
}

func (s *urlService) ShortenURLBatch(ctx context.Context, urls []model.BatchRequest, userID string) ([]model.BatchResponse, error) {
	if len(urls) == 0 {
		return nil, ErrEmptyURLBatch
	}

	records := make([]model.URLRecord, len(urls))
	for i := range urls {
		if urls[i].OriginalURL == "" {
			return nil, ErrEmptyURL
		}
		records[i] = model.URLRecord{
			ShortURL:    s.generateShortURL(),
			OriginalURL: urls[i].OriginalURL,
			UserID:      userID,
		}
	}

	savedRecords, err := s.repo.SaveBatch(ctx, records)
	if err != nil {
		return nil, err
	}

	resp := make([]model.BatchResponse, len(urls))
	for i := range savedRecords {
		resp[i] = model.BatchResponse{
			CorrelationID: urls[i].CorrelationID,
			ShortURL:      fmt.Sprintf("%s/%s", s.baseURL, savedRecords[i].ShortURL),
		}
	}

	return resp, nil
}

func (s *urlService) GetOriginalURL(ctx context.Context, shortID string) (string, error) {
	if shortID == "" {
		return "", ErrEmptyShortID
	}

	url, err := s.repo.GetByShortURL(ctx, shortID)
	if errors.Is(err, repository.ErrURLNotFound) {
		return "", fmt.Errorf("short ID '%s' not found: %w", shortID, ErrURLNotFound)
	}
	if errors.Is(err, repository.ErrURLDeleted) {
		return "", fmt.Errorf("short ID '%s' has been deleted: %w", shortID, ErrURLDeleted)
	}

	return url.OriginalURL, nil
}

func (s *urlService) GetUserURLs(ctx context.Context, userID string) ([]model.UserURLResponse, error) {
	records, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	result := make([]model.UserURLResponse, len(records))
	for i, record := range records {
		result[i] = model.UserURLResponse{
			ShortURL:    fmt.Sprintf("%s/%s", s.baseURL, record.ShortURL),
			OriginalURL: record.OriginalURL,
		}
	}

	return result, nil
}

func (s *urlService) DeleteURLs(ctx context.Context, userID string, shortURLs []string) {
	if len(shortURLs) == 0 {
		return
	}

	go func() {
		g := new(errgroup.Group)
		semaphore := make(chan struct{}, deleteMaxConcurrency)

		for batch := range slices.Chunk(shortURLs, deleteBatchSize) {
			g.Go(func() error {
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				return s.repo.DeleteURLs(ctx, batch, userID)
			})
		}

		if err := g.Wait(); err != nil {
			s.logger.Error("не удалось удалить пачкой урлы", zap.Error(err))
		}
	}()
}

func (s *urlService) Ping(ctx context.Context) error {
	return s.repo.Ping(ctx)
}

func (s *urlService) generateShortURL() string {
	b := make([]byte, 6)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:8]
}
