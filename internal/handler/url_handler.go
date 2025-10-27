package handler

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/Gustik/shortener/internal/service"
	"github.com/go-chi/chi/v5"
)

type URLHandler struct {
	service service.URLService
}

func NewURLHandler(service service.URLService) *URLHandler {
	return &URLHandler{
		service: service,
	}
}

func (h *URLHandler) ShortenURL(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	shortURL, err := h.service.ShortenURL(strings.TrimSpace(string(body)))
	if errors.Is(err, service.ErrEmptyURL) {
		http.Error(w, "URL cannot be empty", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, "Failed to shorten URL", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

func (h *URLHandler) GetOriginalURL(w http.ResponseWriter, r *http.Request) {
	shortID := chi.URLParam(r, "id")

	originalURL, err := h.service.GetOriginalURL(shortID)
	if errors.Is(err, service.ErrURLNotFound) {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func GetShortID(path string) string {
	return strings.TrimPrefix(path, "/")
}
