package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Gustik/shortener/internal/handler"
	"github.com/Gustik/shortener/internal/model"
	"github.com/Gustik/shortener/internal/repository"
	"github.com/Gustik/shortener/internal/service"
	"github.com/Gustik/shortener/internal/zaplog"

	"github.com/stretchr/testify/assert"
)

const (
	baseURL   = "http://localhost:8080"
	jwtSecret = "test-secret-key"
)

func TestURLHandler_ShortenURL(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		contentType  string
		body         string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "Успешное создание урла",
			method:       http.MethodPost,
			contentType:  "text/plain",
			body:         "https://ya.ru",
			expectedCode: http.StatusCreated,
			expectedBody: baseURL,
		},
		{
			name:         "Неправильный content type",
			method:       http.MethodPost,
			contentType:  "application/json",
			body:         "https://ya.ru",
			expectedCode: http.StatusBadRequest,
			expectedBody: "Invalid content type",
		},
		{
			name:         "Пустое тело запроса",
			method:       http.MethodPost,
			contentType:  "text/plain",
			body:         "",
			expectedCode: http.StatusBadRequest,
			expectedBody: "URL cannot be empty",
		},
	}

	repo := repository.NewInMemoryURLRepository()
	service := service.NewURLService(repo, baseURL, zaplog.NewNoop())
	router := handler.SetupRoutes(handler.NewURLHandler(context.Background(), service, zaplog.NewNoop()), jwtSecret)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, "/", bytes.NewBufferString(tt.body))
			r.Header.Set("Content-Type", tt.contentType)

			w := httptest.NewRecorder()

			router.ServeHTTP(w, r)

			assert.Equal(t, tt.expectedCode, w.Code, "Код не тот что ждем")

			if tt.expectedBody != "" {
				if tt.expectedCode == http.StatusCreated {
					assert.True(t, strings.HasPrefix(w.Body.String(), tt.expectedBody), "Префикс не тот")
				} else {
					assert.Contains(t, w.Body.String(), tt.expectedBody)
				}
			}
		})
	}
}

func TestURLHandler_ShortenURLV2(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		contentType  string
		body         string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "Успешное создание урла",
			method:       http.MethodPost,
			contentType:  "application/json",
			body:         `{"url": "https://ya.ru"}`,
			expectedCode: http.StatusCreated,
			expectedBody: baseURL,
		},
		{
			name:         "Неправильный content type",
			method:       http.MethodPost,
			contentType:  "text/plain",
			body:         `{"url": "https://ya.ru"}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: "Invalid content type",
		},
		{
			name:         "Невалидный json",
			method:       http.MethodPost,
			contentType:  "application/json",
			body:         "{invalid json",
			expectedCode: http.StatusBadRequest,
			expectedBody: "Failed to decode json",
		},
		{
			name:         "Пустой url",
			method:       http.MethodPost,
			contentType:  "application/json",
			body:         `{"url": ""}`,
			expectedCode: http.StatusBadRequest,
			expectedBody: "URL cannot be empty",
		},
	}

	repo := repository.NewInMemoryURLRepository()
	service := service.NewURLService(repo, baseURL, zaplog.NewNoop())
	router := handler.SetupRoutes(handler.NewURLHandler(context.Background(), service, zaplog.NewNoop()), jwtSecret)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, "/api/shorten", bytes.NewBufferString(tt.body))
			r.Header.Set("Content-Type", tt.contentType)

			w := httptest.NewRecorder()

			router.ServeHTTP(w, r)

			assert.Equal(t, tt.expectedCode, w.Code, "Код не тот что ждем")

			if tt.expectedBody != "" {
				if tt.expectedCode == http.StatusCreated {
					var resp model.Response
					if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
						t.Fatal("Не удалось декодировать ответ")
					}
					assert.True(t, strings.HasPrefix(resp.Result, tt.expectedBody), "Префикс не тот")
				} else {
					assert.Contains(t, w.Body.String(), tt.expectedBody)
				}
			}
		})
	}
}

func TestURLHandler_ShortenURLBatch(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		contentType  string
		body         string
		expectedCode int
		expectedBody string
		checkResult  bool
	}{
		{
			name:        "Успешное создание batch URLs",
			method:      http.MethodPost,
			contentType: "application/json",
			body: `[
				{"correlation_id": "req-1", "original_url": "https://practicum.yandex.ru"},
				{"correlation_id": "req-2", "original_url": "https://yandex.ru"},
				{"correlation_id": "req-3", "original_url": "https://github.com"}
			]`,
			expectedCode: http.StatusCreated,
			checkResult:  true,
		},
		{
			name:         "Неправильный content type",
			method:       http.MethodPost,
			contentType:  "text/plain",
			body:         `[{"correlation_id": "req-1", "original_url": "https://ya.ru"}]`,
			expectedCode: http.StatusBadRequest,
			expectedBody: "Invalid content type",
		},
		{
			name:         "Невалидный json",
			method:       http.MethodPost,
			contentType:  "application/json",
			body:         "{invalid json",
			expectedCode: http.StatusBadRequest,
			expectedBody: "Failed to decode json",
		},
		{
			name:         "Пустой массив",
			method:       http.MethodPost,
			contentType:  "application/json",
			body:         `[]`,
			expectedCode: http.StatusBadRequest,
			expectedBody: "URL batch cannot be empty",
		},
		{
			name:         "Один элемент в массиве",
			method:       http.MethodPost,
			contentType:  "application/json",
			body:         `[{"correlation_id": "single", "original_url": "https://example.com"}]`,
			expectedCode: http.StatusCreated,
			checkResult:  true,
		},
	}

	repo := repository.NewInMemoryURLRepository()
	service := service.NewURLService(repo, baseURL, zaplog.NewNoop())
	router := handler.SetupRoutes(handler.NewURLHandler(context.Background(), service, zaplog.NewNoop()), jwtSecret)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, "/api/shorten/batch", bytes.NewBufferString(tt.body))
			r.Header.Set("Content-Type", tt.contentType)

			w := httptest.NewRecorder()

			router.ServeHTTP(w, r)

			assert.Equal(t, tt.expectedCode, w.Code, "Код не тот что ждем")

			if tt.checkResult && tt.expectedCode == http.StatusCreated {
				var resp []model.BatchResponse
				if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
					t.Fatal("Не удалось декодировать ответ")
				}

				var req []model.BatchRequest
				if err := json.Unmarshal([]byte(tt.body), &req); err != nil {
					t.Fatal("Не удалось декодировать входной запрос")
				}

				assert.Equal(t, len(req), len(resp), "Количество элементов в ответе должно совпадать с запросом")

				for i, item := range resp {
					assert.Equal(t, req[i].CorrelationID, item.CorrelationID, "CorrelationID должен совпадать")
					assert.True(t, strings.HasPrefix(item.ShortURL, baseURL), "ShortURL должен начинаться с baseURL")
					assert.NotEmpty(t, item.ShortURL, "ShortURL не должен быть пустым")
				}
			} else if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
		})
	}
}

func TestURLHandler_GetOriginalURL(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		url          string
		method       string
		expectedCode int
		expectedBody string
		expectedURL  string
	}{
		{
			name:         "Не существующие url",
			path:         "/notexists",
			method:       http.MethodGet,
			expectedCode: http.StatusNotFound,
			expectedBody: "URL not found",
		},
		{
			name:         "Успешно нашел url",
			path:         "/shortID1",
			url:          "https://ya.ru",
			method:       http.MethodGet,
			expectedCode: http.StatusTemporaryRedirect,
			expectedURL:  "https://ya.ru",
		},
	}

	repo := repository.NewInMemoryURLRepository()
	service := service.NewURLService(repo, baseURL, zaplog.NewNoop())
	router := handler.SetupRoutes(handler.NewURLHandler(context.Background(), service, zaplog.NewNoop()), jwtSecret)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedCode == http.StatusTemporaryRedirect {
				repo.Save(context.Background(), strings.TrimPrefix(tt.path, "/"), tt.url, "test-user")
			}

			r := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, r)

			assert.Equal(t, tt.expectedCode, w.Code)

			if tt.expectedCode == http.StatusTemporaryRedirect {
				assert.Equal(t, tt.expectedURL, w.Header().Get("Location"))
			} else {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
		})
	}
}

func TestURLHandler_GetOriginalURL_Deleted(t *testing.T) {
	repo := repository.NewInMemoryURLRepository()
	service := service.NewURLService(repo, baseURL, zaplog.NewNoop())
	router := handler.SetupRoutes(handler.NewURLHandler(context.Background(), service, zaplog.NewNoop()), jwtSecret)

	ctx := context.Background()
	userID := "test-user"
	shortID := "deletedURL"

	// Создаём URL
	_, err := repo.Save(ctx, shortID, "https://example.com", userID)
	assert.NoError(t, err)

	// Удаляем URL
	err = repo.DeleteURLs(ctx, []string{shortID}, userID)
	assert.NoError(t, err)

	// Пытаемся получить удалённый URL
	r := httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusGone, w.Code)
	assert.Contains(t, w.Body.String(), "URL has been deleted")
}

func TestURLHandler_DeleteUserURLs(t *testing.T) {
	tests := []struct {
		name         string
		method       string
		contentType  string
		body         string
		expectedCode int
		expectedBody string
	}{
		{
			name:         "Успешное удаление URLs",
			method:       http.MethodDelete,
			contentType:  "application/json",
			body:         `["url1", "url2", "url3"]`,
			expectedCode: http.StatusAccepted,
		},
		{
			name:         "Неправильный content type",
			method:       http.MethodDelete,
			contentType:  "text/plain",
			body:         `["url1"]`,
			expectedCode: http.StatusBadRequest,
			expectedBody: "Invalid content type",
		},
		{
			name:         "Невалидный json",
			method:       http.MethodDelete,
			contentType:  "application/json",
			body:         "{invalid json",
			expectedCode: http.StatusBadRequest,
			expectedBody: "Failed to decode json",
		},
		{
			name:         "Пустой массив",
			method:       http.MethodDelete,
			contentType:  "application/json",
			body:         `[]`,
			expectedCode: http.StatusBadRequest,
			expectedBody: "Empty URL list",
		},
	}

	repo := repository.NewInMemoryURLRepository()
	service := service.NewURLService(repo, baseURL, zaplog.NewNoop())
	router := handler.SetupRoutes(handler.NewURLHandler(context.Background(), service, zaplog.NewNoop()), jwtSecret)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, "/api/user/urls", bytes.NewBufferString(tt.body))
			r.Header.Set("Content-Type", tt.contentType)

			w := httptest.NewRecorder()

			router.ServeHTTP(w, r)

			assert.Equal(t, tt.expectedCode, w.Code, "Код не тот что ждем")

			if tt.expectedBody != "" {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
		})
	}
}
