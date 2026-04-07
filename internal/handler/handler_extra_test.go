package handler_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Gustik/shortener/internal/audit"
	"github.com/Gustik/shortener/internal/handler"
	"github.com/Gustik/shortener/internal/handler/middleware"
	"github.com/Gustik/shortener/internal/repository"
	"github.com/Gustik/shortener/internal/service"
	"github.com/Gustik/shortener/internal/zaplog"
)

func makeJWT(userID, secret string) string {
	claims := &middleware.Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, _ := token.SignedString([]byte(secret))
	return s
}

// --- GetUserURLs ---

func TestURLHandler_GetUserURLs(t *testing.T) {
	ctx := context.Background()
	const userID = "test-user-1"

	t.Run("Возвращает URL пользователя", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()
		_, err := repo.Save(ctx, "abc12345", "https://ya.ru", userID)
		require.NoError(t, err)
		_, err = repo.Save(ctx, "xyz99999", "https://google.com", userID)
		require.NoError(t, err)

		svc := service.NewURLService(repo, baseURL, zaplog.NewNoop())
		router := handler.SetupRoutes(handler.NewURLHandler(ctx, svc, zaplog.NewNoop(), audit.DummyPublisher{}), jwtSecret, nil)

		r := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
		r.AddCookie(&http.Cookie{Name: "token", Value: makeJWT(userID, jwtSecret)})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "https://ya.ru")
		assert.Contains(t, w.Body.String(), "https://google.com")
	})

	t.Run("Пользователь без URL возвращает 204", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()
		svc := service.NewURLService(repo, baseURL, zaplog.NewNoop())
		router := handler.SetupRoutes(handler.NewURLHandler(ctx, svc, zaplog.NewNoop(), audit.DummyPublisher{}), jwtSecret, nil)

		r := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
		r.AddCookie(&http.Cookie{Name: "token", Value: makeJWT(userID, jwtSecret)})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})

	t.Run("URL другого пользователя не попадает в ответ", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()
		_, err := repo.Save(ctx, "other1", "https://other.com", "other-user")
		require.NoError(t, err)

		svc := service.NewURLService(repo, baseURL, zaplog.NewNoop())
		router := handler.SetupRoutes(handler.NewURLHandler(ctx, svc, zaplog.NewNoop(), audit.DummyPublisher{}), jwtSecret, nil)

		r := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
		r.AddCookie(&http.Cookie{Name: "token", Value: makeJWT(userID, jwtSecret)})
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}

// --- Ping ---

func TestURLHandler_Ping(t *testing.T) {
	repo := repository.NewInMemoryURLRepository()
	svc := service.NewURLService(repo, baseURL, zaplog.NewNoop())
	router := handler.SetupRoutes(handler.NewURLHandler(context.Background(), svc, zaplog.NewNoop(), audit.DummyPublisher{}), jwtSecret, nil)

	r := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
}

// --- GetStats ---

func TestURLHandler_GetStats(t *testing.T) {
	ctx := context.Background()

	_, trustedNet, err := net.ParseCIDR("127.0.0.1/32")
	require.NoError(t, err)

	t.Run("Из доверенной подсети возвращает статистику", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()
		_, err := repo.Save(ctx, "s1", "https://ya.ru", "u1")
		require.NoError(t, err)
		_, err = repo.Save(ctx, "s2", "https://google.com", "u2")
		require.NoError(t, err)

		svc := service.NewURLService(repo, baseURL, zaplog.NewNoop())
		router := handler.SetupRoutes(handler.NewURLHandler(ctx, svc, zaplog.NewNoop(), audit.DummyPublisher{}), jwtSecret, trustedNet)

		r := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
		r.Header.Set("X-Real-IP", "127.0.0.1")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"urls"`)
		assert.Contains(t, w.Body.String(), `"users"`)
	})

	t.Run("Без доверенной подсети возвращает 403", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()
		svc := service.NewURLService(repo, baseURL, zaplog.NewNoop())
		router := handler.SetupRoutes(handler.NewURLHandler(ctx, svc, zaplog.NewNoop(), audit.DummyPublisher{}), jwtSecret, nil)

		r := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("Из недоверенного IP возвращает 403", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()
		svc := service.NewURLService(repo, baseURL, zaplog.NewNoop())
		router := handler.SetupRoutes(handler.NewURLHandler(ctx, svc, zaplog.NewNoop(), audit.DummyPublisher{}), jwtSecret, trustedNet)

		r := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
		r.Header.Set("X-Real-IP", "10.0.0.1")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

// --- GetShortID ---

func TestGetShortID(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/abc12345", "abc12345"},
		{"/", ""},
		{"abc12345", "abc12345"},
	}

	for _, tt := range tests {
		result := handler.GetShortID(tt.path)
		assert.Equal(t, tt.expected, result, "path: %s", tt.path)
	}
}

// --- ShortenURL redirect flow ---

func TestURLHandler_ShortenAndRedirect(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewInMemoryURLRepository()
	svc := service.NewURLService(repo, baseURL, zaplog.NewNoop())
	router := handler.SetupRoutes(handler.NewURLHandler(ctx, svc, zaplog.NewNoop(), audit.DummyPublisher{}), jwtSecret, nil)

	// Создаём короткий URL
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://ya.ru"))
	r.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	require.Equal(t, http.StatusCreated, w.Code)
	shortURL := strings.TrimSpace(w.Body.String())
	shortID := strings.TrimPrefix(shortURL, baseURL+"/")

	// Редирект по короткому ID
	r2 := httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, r2)

	assert.Equal(t, http.StatusTemporaryRedirect, w2.Code)
	assert.Equal(t, "https://ya.ru", w2.Header().Get("Location"))
}
