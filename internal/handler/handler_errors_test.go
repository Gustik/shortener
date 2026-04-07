package handler_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Gustik/shortener/internal/audit"
	"github.com/Gustik/shortener/internal/handler"
	"github.com/Gustik/shortener/internal/model"
	"github.com/Gustik/shortener/internal/zaplog"
)

// errService — стаб service.URLService, всегда возвращающий ошибку.
// Используется для тестирования 5xx-путей в хендлерах.
type errService struct{}

var errInternal = errors.New("internal error")

func (errService) ShortenURL(_ context.Context, _, _ string) (string, error) {
	return "", errInternal
}
func (errService) ShortenURLBatch(_ context.Context, _ []model.BatchRequest, _ string) ([]model.BatchResponse, error) {
	return nil, errInternal
}
func (errService) GetOriginalURL(_ context.Context, _ string) (string, error) {
	return "", errInternal
}
func (errService) GetUserURLs(_ context.Context, _ string) ([]model.UserURLResponse, error) {
	return nil, errInternal
}
func (errService) DeleteURLs(_ context.Context, _ string, _ []string) {}
func (errService) Stats(_ context.Context) (int, int, error)           { return 0, 0, errInternal }
func (errService) Ping(_ context.Context) error                        { return errInternal }

func newErrRouter() http.Handler {
	h := handler.NewURLHandler(context.Background(), errService{}, zaplog.NewNoop(), audit.DummyPublisher{})
	return handler.SetupRoutes(h, jwtSecret, nil)
}

func TestURLHandler_ShortenURL_ServiceError(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString("https://ya.ru"))
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	newErrRouter().ServeHTTP(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestURLHandler_ShortenURLV2_ServiceError(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBufferString(`{"url":"https://ya.ru"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newErrRouter().ServeHTTP(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestURLHandler_ShortenURLBatch_ServiceError(t *testing.T) {
	body := `[{"correlation_id":"c1","original_url":"https://ya.ru"}]`
	r := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	newErrRouter().ServeHTTP(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestURLHandler_GetOriginalURL_ServiceError(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/abc12345", nil)
	w := httptest.NewRecorder()
	newErrRouter().ServeHTTP(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestURLHandler_GetUserURLs_ServiceError(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	r.AddCookie(&http.Cookie{Name: "token", Value: makeJWT("user1", jwtSecret)})
	w := httptest.NewRecorder()
	newErrRouter().ServeHTTP(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestURLHandler_Ping_ServiceError(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	newErrRouter().ServeHTTP(w, r)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
