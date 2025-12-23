package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	myMiddleware "github.com/Gustik/shortener/internal/handler/middleware"
)

func SetupRoutes(handler *URLHandler, jwtSecret string) http.Handler {
	r := chi.NewRouter()

	r.Use(myMiddleware.RequestLogger(handler.logger))
	r.Use(myMiddleware.GzipMiddleware(handler.logger))
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(myMiddleware.AuthMiddleware(jwtSecret, handler.logger))

	r.With(myMiddleware.ContentTypeMiddleware("text/plain")).Post("/", handler.ShortenURL)
	r.With(myMiddleware.ContentTypeMiddleware("application/json")).Post("/api/shorten", handler.ShortenURLV2)
	r.With(myMiddleware.ContentTypeMiddleware("application/json")).Post("/api/shorten/batch", handler.ShortenURLBatch)
	r.Get("/{id}", handler.GetOriginalURL)
	r.Get("/api/user/urls", handler.GetUserURLs)
	r.Get("/ping", handler.Ping)

	return r
}
