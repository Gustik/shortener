package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	myMiddleware "github.com/Gustik/shortener/internal/handler/middleware"
)

func SetupRoutes(handler *URLHandler) http.Handler {
	r := chi.NewRouter()

	r.Use(myMiddleware.RequestLogger)
	r.Use(myMiddleware.GzipMiddleware)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)

	r.With(myMiddleware.ContentTypeMiddleware("text/plain")).Post("/", handler.ShortenURL)
	r.With(myMiddleware.ContentTypeMiddleware("application/json")).Post("/api/shorten", handler.ShortenURLV2)
	r.Get("/{id}", handler.GetOriginalURL)

	return r
}
