package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Gustik/shortener/internal/handler/middleware"
	"github.com/Gustik/shortener/internal/zaplog"
)

const testJWTSecret = "test-secret"

func makeJWT(userID string) string {
	claims := &middleware.Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, _ := token.SignedString([]byte(testJWTSecret))
	return s
}

func TestAuthMiddleware_NoToken(t *testing.T) {
	var capturedUserID string

	handler := middleware.AuthMiddleware(testJWTSecret, zaplog.NewNoop())(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uid, ok := middleware.GetUserID(r.Context())
			require.True(t, ok, "userID должен быть в контексте")
			capturedUserID = uid
			w.WriteHeader(http.StatusOK)
		}),
	)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, capturedUserID, "должен быть сгенерирован новый userID")

	// Должна быть установлена кука с JWT
	resp := w.Result()
	defer resp.Body.Close()
	cookies := resp.Cookies()
	require.NotEmpty(t, cookies)
	var tokenCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "token" {
			tokenCookie = c
			break
		}
	}
	require.NotNil(t, tokenCookie, "должна быть установлена кука token")
	assert.NotEmpty(t, tokenCookie.Value)
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	const userID = "user-123"
	var capturedUserID string

	handler := middleware.AuthMiddleware(testJWTSecret, zaplog.NewNoop())(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uid, _ := middleware.GetUserID(r.Context())
			capturedUserID = uid
			w.WriteHeader(http.StatusOK)
		}),
	)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: "token", Value: makeJWT(userID)})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, userID, capturedUserID)
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	var capturedUserID string

	handler := middleware.AuthMiddleware(testJWTSecret, zaplog.NewNoop())(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uid, _ := middleware.GetUserID(r.Context())
			capturedUserID = uid
			w.WriteHeader(http.StatusOK)
		}),
	)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(&http.Cookie{Name: "token", Value: "not.a.valid.jwt"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code)
	// Невалидный токен → генерируется новый userID (не пустой и не "user-123")
	assert.NotEmpty(t, capturedUserID)
}

func TestGetUserID(t *testing.T) {
	t.Run("Возвращает userID из контекста", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), middleware.UserIDContextKey, "user-abc")

		uid, ok := middleware.GetUserID(ctx)

		assert.True(t, ok)
		assert.Equal(t, "user-abc", uid)
	})

	t.Run("Возвращает false если userID отсутствует", func(t *testing.T) {
		uid, ok := middleware.GetUserID(context.Background())

		assert.False(t, ok)
		assert.Empty(t, uid)
	})
}
