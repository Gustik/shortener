package handler

import (
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
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	originalURL := strings.TrimSpace(string(body))
	if originalURL == "" {
		http.Error(w, "URL cannot be empty", http.StatusBadRequest)
		return
	}

	shortURL, err := h.service.ShortenURL(originalURL)
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
	if shortID == "" {
		http.Error(w, "Short ID is required", http.StatusBadRequest)
		return
	}

	originalURL, err := h.service.GetOriginalURL(shortID)
	if err != nil {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func GetShortID(path string) string {
	return strings.TrimPrefix(path, "/")
}
