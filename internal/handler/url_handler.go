package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"github.com/Gustik/shortener/internal/handler/middleware"
	"github.com/Gustik/shortener/internal/model"
	"github.com/Gustik/shortener/internal/service"
	"github.com/go-chi/chi/v5"
)

type URLHandler struct {
	appCtx  context.Context
	service service.URLService
	logger  *zap.Logger
}

func NewURLHandler(appCtx context.Context, service service.URLService, logger *zap.Logger) *URLHandler {
	return &URLHandler{
		appCtx:  appCtx,
		service: service,
		logger:  logger,
	}
}

func (h *URLHandler) ShortenURL(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	shortURL, err := h.service.ShortenURL(r.Context(), strings.TrimSpace(string(body)), userID)
	if errors.Is(err, service.ErrEmptyURL) {
		http.Error(w, "URL cannot be empty", http.StatusBadRequest)
		return
	}

	if errors.Is(err, service.ErrURLExists) {
		w.WriteHeader(http.StatusConflict)
	} else if err != nil {
		h.logger.Error("failed to shorten URL", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	} else {
		w.WriteHeader(http.StatusCreated)
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(shortURL))
}

func (h *URLHandler) ShortenURLV2(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req model.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Failed to decode json", http.StatusBadRequest)
		return
	}

	shortURL, err := h.service.ShortenURL(r.Context(), req.URL, userID)
	if errors.Is(err, service.ErrEmptyURL) {
		http.Error(w, "URL cannot be empty", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if errors.Is(err, service.ErrURLExists) {
		w.WriteHeader(http.StatusConflict)
	} else if err != nil {
		h.logger.Error("failed to shorten URL", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	} else {
		w.WriteHeader(http.StatusCreated)
	}

	resp := model.Response{
		Result: shortURL,
	}

	if err := json.NewEncoder(w).Encode(&resp); err != nil {
		h.logger.Error("failed to encode response", zap.Error(err))
	}
}

func (h *URLHandler) ShortenURLBatch(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req []model.BatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Failed to decode json", http.StatusBadRequest)
		return
	}

	resp, err := h.service.ShortenURLBatch(r.Context(), req, userID)
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

	if errors.Is(err, service.ErrURLDeleted) {
		http.Error(w, "URL has been deleted", http.StatusGone)
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

func (h *URLHandler) GetUserURLs(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	urls, err := h.service.GetUserURLs(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to get user URLs", zap.Error(err))
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	if len(urls) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(urls); err != nil {
		h.logger.Error("failed to encode response", zap.Error(err))
	}
}

func (h *URLHandler) DeleteUserURLs(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var shortURLs []string
	if err := json.NewDecoder(r.Body).Decode(&shortURLs); err != nil {
		http.Error(w, "Failed to decode json", http.StatusBadRequest)
		return
	}

	if len(shortURLs) == 0 {
		http.Error(w, "Empty URL list", http.StatusBadRequest)
		return
	}

	// Асинхронное удаление используя контекст приложения, а не контекст запроса
	h.service.DeleteURLs(h.appCtx, userID, shortURLs)

	w.WriteHeader(http.StatusAccepted)
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
