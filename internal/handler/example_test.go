package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/Gustik/shortener/internal/audit"
	"github.com/Gustik/shortener/internal/handler"
	"github.com/Gustik/shortener/internal/model"
	"github.com/Gustik/shortener/internal/repository"
	"github.com/Gustik/shortener/internal/service"
	"github.com/Gustik/shortener/internal/zaplog"
)

func ExampleURLHandler_ShortenURL() {
	repo := repository.NewInMemoryURLRepository()
	svc := service.NewURLService(repo, baseURL, zaplog.NewNoop())
	router := handler.SetupRoutes(handler.NewURLHandler(context.Background(), svc, zaplog.NewNoop(), audit.DummyPublisher{}), jwtSecret, nil)

	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://practicum.yandex.ru"))
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	fmt.Println(w.Code)
	fmt.Println(strings.HasPrefix(w.Body.String(), baseURL))
	// Output:
	// 201
	// true
}

func ExampleURLHandler_ShortenURLV2() {
	repo := repository.NewInMemoryURLRepository()
	svc := service.NewURLService(repo, baseURL, zaplog.NewNoop())
	router := handler.SetupRoutes(handler.NewURLHandler(context.Background(), svc, zaplog.NewNoop(), audit.DummyPublisher{}), jwtSecret, nil)

	body, _ := json.Marshal(model.Request{URL: "https://practicum.yandex.ru"})
	r := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	var resp model.Response
	json.NewDecoder(w.Body).Decode(&resp)

	fmt.Println(w.Code)
	fmt.Println(strings.HasPrefix(resp.Result, baseURL))
	// Output:
	// 201
	// true
}

func ExampleURLHandler_GetOriginalURL() {
	repo := repository.NewInMemoryURLRepository()
	svc := service.NewURLService(repo, baseURL, zaplog.NewNoop())
	router := handler.SetupRoutes(handler.NewURLHandler(context.Background(), svc, zaplog.NewNoop(), audit.DummyPublisher{}), jwtSecret, nil)

	// Pre-populate a short URL in the repository.
	repo.Save(context.Background(), "abc123", "https://practicum.yandex.ru", "user1")

	r := httptest.NewRequest(http.MethodGet, "/abc123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, r)

	fmt.Println(w.Code)
	fmt.Println(w.Header().Get("Location"))
	// Output:
	// 307
	// https://practicum.yandex.ru
}
