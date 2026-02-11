package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type contextKey string

const UserIDContextKey contextKey = "userID"

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func AuthMiddleware(jwtSecret string, logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("token")

			var userID string

			if err == nil {
				// Кука существует, проверяем токен
				token, err := jwt.ParseWithClaims(cookie.Value, &Claims{}, func(token *jwt.Token) (interface{}, error) {
					return []byte(jwtSecret), nil
				})

				if err == nil && token.Valid {
					// Токен валидный, извлекаем userID
					if claims, ok := token.Claims.(*Claims); ok {
						userID = claims.UserID
						logger.Debug("Валидный JWT токен", zap.String("userID", userID))
					}
				} else {
					logger.Debug("Невалидный JWT токен", zap.Error(err))
				}
			}

			// Если userID пустой (нет куки или токен невалидный), генерируем новый
			if userID == "" {
				userID = uuid.New().String()
				logger.Debug("Генерация нового userID", zap.String("userID", userID))

				// Создаём новый JWT токен
				claims := &Claims{
					UserID: userID,
					RegisteredClaims: jwt.RegisteredClaims{
						ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour * 365)), // 1 год
					},
				}

				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				tokenString, err := token.SignedString([]byte(jwtSecret))
				if err != nil {
					logger.Error("Ошибка создания JWT токена", zap.Error(err))
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
					return
				}

				// Устанавливаем куку
				http.SetCookie(w, &http.Cookie{
					Name:     "token",
					Value:    tokenString,
					Path:     "/",
					HttpOnly: true,
					MaxAge:   365 * 24 * 60 * 60, // 1 год в секундах
				})
			}

			// Добавляем userID в контекст
			ctx := context.WithValue(r.Context(), UserIDContextKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID извлекает userID из контекста
func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDContextKey).(string)
	return userID, ok
}
