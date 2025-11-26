package main

import (
	"log"
	"net/http"

	"github.com/Gustik/shortener/internal/config"
	"github.com/Gustik/shortener/internal/handler"
	"github.com/Gustik/shortener/internal/repository"
	"github.com/Gustik/shortener/internal/service"
	"github.com/Gustik/shortener/internal/zaplog"
)

func main() {
	var err error
	cfg := config.Load()

	logger, err := zaplog.New(cfg.LogLevel)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	var repo repository.URLRepository

	if cfg.StorageType == config.StorageFile {
		repo, err = repository.NewFileURLRepository(cfg.FileStoragePath)
		if err != nil {
			logger.Sugar().Fatalf("Ошибка инициализации репозитория: %v", err)
		}
	} else {
		repo = repository.NewInMemoryURLRepository()
	}

	svc := service.NewURLService(repo, cfg.BaseURL)
	h := handler.NewURLHandler(svc, logger)

	router := handler.SetupRoutes(h)

	logger.Sugar().Infof("Запускаем сервер по адресу %s", cfg.ServerAddress.String())
	if err := http.ListenAndServe(cfg.ServerAddress.String(), router); err != nil {
		logger.Sugar().Fatalf("Ошибка при запуске: %v", err)
	}
}
