package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Gustik/shortener/internal/handler/middleware"
	"github.com/Gustik/shortener/internal/zaplog"
)

// --- ContentTypeMiddleware ---

func TestContentTypeMiddleware_Correct(t *testing.T) {
	called := false
	handler := middleware.ContentTypeMiddleware("application/json")(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusOK)
		}),
	)

	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.True(t, called, "следующий хендлер должен быть вызван")
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestContentTypeMiddleware_Wrong(t *testing.T) {
	called := false
	handler := middleware.ContentTypeMiddleware("application/json")(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
		}),
	)

	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.False(t, called, "следующий хендлер не должен быть вызван")
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid content type")
}

// --- RequestLogger ---

func TestRequestLogger(t *testing.T) {
	called := false
	handler := middleware.RequestLogger(zaplog.NewNoop())(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			called = true
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte("hello"))
		}),
	)

	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.True(t, called)
	assert.Equal(t, http.StatusCreated, w.Code)
}
