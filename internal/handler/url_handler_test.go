package handler_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Gustik/shortener/internal/handler"
	"github.com/Gustik/shortener/internal/repository"
	"github.com/Gustik/shortener/internal/service"

	"github.com/stretchr/testify/assert"
)

const baseURL = "http://localhost:8080"

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
			name:         "Неправильный method",
			method:       http.MethodGet,
			contentType:  "text/plain",
			body:         "https://ya.ru",
			expectedCode: http.StatusMethodNotAllowed,
			expectedBody: "Method not allowed",
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

	repo := repository.NewMockURLRepository()
	service := service.NewURLService(repo, baseURL)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, "/", bytes.NewBufferString(tt.body))
			r.Header.Set("Content-Type", tt.contentType)

			w := httptest.NewRecorder()

			h := handler.NewURLHandler(service)
			h.ShortenURL(w, r)

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

func TestURLHandler_GetOriginalURL(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		url          string
		method       string
		expectedCode int
		expectedBody string
		expectedUrl  string
	}{
		{
			name:         "Неправильный method",
			path:         "/somepath",
			method:       http.MethodPost,
			expectedCode: http.StatusMethodNotAllowed,
			expectedBody: "Method not allowed",
		},
		{
			name:         "Пустой shortID",
			path:         "/",
			method:       http.MethodGet,
			expectedCode: http.StatusBadRequest,
			expectedBody: "Short ID is required",
		},
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
			expectedUrl:  "https://ya.ru",
		},
	}

	repo := repository.NewMockURLRepository()
	service := service.NewURLService(repo, baseURL)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedCode == http.StatusTemporaryRedirect {
				repo.Save(handler.GetShortID(tt.path), tt.url)
			}

			h := handler.NewURLHandler(service)
			r := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			h.GetOriginalURL(w, r)

			assert.Equal(t, tt.expectedCode, w.Code)

			if tt.expectedCode == http.StatusTemporaryRedirect {
				assert.Equal(t, tt.expectedUrl, w.Header().Get("Location"))
			} else {
				assert.Contains(t, w.Body.String(), tt.expectedBody)
			}
		})
	}
}
