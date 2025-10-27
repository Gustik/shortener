package main

import (
	"log"
	"net/http"

	"github.com/Gustik/shortener/internal/config"
	"github.com/Gustik/shortener/internal/handler"
	"github.com/Gustik/shortener/internal/repository"
	"github.com/Gustik/shortener/internal/service"
)

func main() {
	cfg := config.Load()

	repo := repository.NewInMemoryURLRepository()
	svc := service.NewURLService(repo, cfg.BaseURL)
	h := handler.NewURLHandler(svc)

	router := handler.SetupRoutes(h)

	log.Printf("Запускаем сервер по адресу %s", cfg.ServerAddress.String())
	if err := http.ListenAndServe(cfg.ServerAddress.String(), router); err != nil {
		log.Fatalf("Ошибка при запуске: %v", err)
	}
}
