package service

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/Gustik/shortener/internal/repository"
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
	shortID := s.generateShortID()

	err := s.repo.Save(shortID, originalURL)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s", s.baseURL, shortID), nil
}

func (s *urlService) GetOriginalURL(shortID string) (string, error) {
	return s.repo.GetByID(shortID)
}

func (s *urlService) generateShortID() string {
	b := make([]byte, 6)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:8]
}
