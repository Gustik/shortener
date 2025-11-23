package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/Gustik/shortener/internal/logger"
	"github.com/Gustik/shortener/internal/model"
	"github.com/Gustik/shortener/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
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

	shortURL, err := h.service.ShortenURL(r.Context(), strings.TrimSpace(string(body)))
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

func (h *URLHandler) ShortenURLV2(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req model.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Failed to decode json", http.StatusBadRequest)
		return
	}

	shortURL, err := h.service.ShortenURL(r.Context(), req.URL)
	if errors.Is(err, service.ErrEmptyURL) {
		http.Error(w, "URL cannot be empty", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, "Failed to shorten URL", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	resp := model.Response{
		Result: shortURL,
	}

	if err := json.NewEncoder(w).Encode(&resp); err != nil {
		logger.Log.Error("failed to encode response", zap.Error(err))
	}
}

func (h *URLHandler) GetOriginalURL(w http.ResponseWriter, r *http.Request) {
	shortID := chi.URLParam(r, "id")

	originalURL, err := h.service.GetOriginalURL(r.Context(), shortID)
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
