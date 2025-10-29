package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func ContentTypeMiddleware(contentType string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Content-Type") != contentType {
				http.Error(w, "Invalid content type", http.StatusBadRequest)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func SetupRoutes(handler *URLHandler) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.Logger)    // логирование запросов
	r.Use(middleware.Recoverer) // восстановление после panic
	r.Use(middleware.RequestID) // ID для каждого запроса
	r.Use(middleware.RealIP)    // получение реального IP

	r.With(ContentTypeMiddleware("text/plain")).Post("/", handler.ShortenURL)
	r.Get("/{id}", handler.GetOriginalURL)

	return r
}
