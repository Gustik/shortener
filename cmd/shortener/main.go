package main

import (
	"net/http"

	"github.com/Gustik/shortener/internal/config"
	"github.com/Gustik/shortener/internal/handler"
	"github.com/Gustik/shortener/internal/logger"
	"github.com/Gustik/shortener/internal/repository"
	"github.com/Gustik/shortener/internal/service"
)

func main() {
	cfg := config.Load()

	logger.Initialize(cfg.LogLevel)

	repo := repository.NewInMemoryURLRepository()
	svc := service.NewURLService(repo, cfg.BaseURL)
	h := handler.NewURLHandler(svc)

	router := handler.SetupRoutes(h)

	logger.Log.Sugar().Infof("Запускаем сервер по адресу %s", cfg.ServerAddress.String())
	if err := http.ListenAndServe(cfg.ServerAddress.String(), router); err != nil {
		logger.Log.Sugar().Fatalf("Ошибка при запуске: %v", err)
	}
}
