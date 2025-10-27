package service

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/Gustik/shortener/internal/repository"
)

var (
	ErrEmptyURL     = errors.New("URL cannot be empty")
	ErrEmptyShortID = errors.New("ShortID cannot be empty")
	ErrURLNotFound  = errors.New("URL not found")
)

type URLService interface {
	ShortenURL(originalURL string) (string, error)
	GetOriginalURL(shortID string) (string, error)
}

type urlService struct {
	repo    repository.URLRepository
	baseURL string
}

func NewURLService(repo repository.URLRepository, baseURL string) URLService {
	return &urlService{
		repo:    repo,
		baseURL: baseURL,
	}
}

func (s *urlService) ShortenURL(originalURL string) (string, error) {
	if originalURL == "" {
		return "", ErrEmptyURL
	}

	shortID := s.generateShortID()

	err := s.repo.Save(shortID, originalURL)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s", s.baseURL, shortID), nil
}

func (s *urlService) GetOriginalURL(shortID string) (string, error) {
	if shortID == "" {
		return "", ErrEmptyShortID
	}

	url, err := s.repo.GetByID(shortID)
	if errors.Is(err, repository.ErrURLNotFound) {
		return "", ErrURLNotFound
	}

	return url, nil
}

func (s *urlService) generateShortID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:8]
}
