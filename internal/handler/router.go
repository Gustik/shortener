package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	myMiddleware "github.com/Gustik/shortener/internal/handler/middleware"
)

func SetupRoutes(handler *URLHandler) http.Handler {
	r := chi.NewRouter()

	r.Use(myMiddleware.RequestLogger) // логирование запросов
	r.Use(middleware.Recoverer)       // восстановление после panic
	r.Use(middleware.RequestID)       // ID для каждого запроса
	r.Use(middleware.RealIP)          // получение реального IP

	r.With(myMiddleware.ContentTypeMiddleware("text/plain")).Post("/", handler.ShortenURL)
	r.Get("/{id}", handler.GetOriginalURL)

	return r
}
