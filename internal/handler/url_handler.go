package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/Gustik/shortener/internal/model"
	"github.com/Gustik/shortener/internal/service"
	"github.com/go-chi/chi/v5"
)

type URLHandler struct {
	service service.URLService
	logger  *zap.Logger
}

func NewURLHandler(service service.URLService, logger *zap.Logger) *URLHandler {
	return &URLHandler{
		service: service,
		logger:  logger,
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
		h.logger.Error("failed to shorten URL", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
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
		h.logger.Error("failed to shorten URL", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	resp := model.Response{
		Result: shortURL,
	}

	if err := json.NewEncoder(w).Encode(&resp); err != nil {
		h.logger.Error("failed to encode response", zap.Error(err))
	}
}

func (h *URLHandler) ShortenURLBatch(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var req []model.BatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Failed to decode json", http.StatusBadRequest)
		return
	}

	resp, err := h.service.ShortenURLBatch(r.Context(), req)
	if errors.Is(err, service.ErrEmptyURLBatch) {
		http.Error(w, "URL batch cannot be empty", http.StatusBadRequest)
		return
	}

	if err != nil {
		h.logger.Error("failed to shorten URL batch", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(&resp); err != nil {
		h.logger.Error("failed to encode response", zap.Error(err))
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
		h.logger.Error("failed to get original URL", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}

func (h *URLHandler) Ping(w http.ResponseWriter, r *http.Request) {
	err := h.service.Ping(r.Context())
	if err != nil {
		h.logger.Error("ping failed", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func GetShortID(path string) string {
	return strings.TrimPrefix(path, "/")
}
