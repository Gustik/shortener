package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5"

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
		log.Fatalf("Ошибка инициализации логгера: %v", err)
	}
	defer logger.Sync()

	var repo repository.URLRepository

	switch cfg.StorageType {
	case config.StorageFile:
		file, err := os.OpenFile(cfg.FileStoragePath, os.O_RDWR|os.O_CREATE, 0666)
		if err != nil {
			log.Fatalf("Ошибка открытия файла репозитория: %v", err)
		}
		repo, err = repository.NewFileURLRepository(file)
		if err != nil {
			logger.Sugar().Fatalf("Ошибка инициализации репозитория: %v", err)
		}
		defer file.Close()
	case config.StorageSQL:
		conn, err := pgx.Connect(context.Background(), cfg.DatabaseDSN)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
			os.Exit(1)
		}
		defer conn.Close(context.Background())

		repo, err = repository.NewSQLRepository(conn)
	default:
		repo = repository.NewInMemoryURLRepository()
	}

	svc := service.NewURLService(repo, cfg.BaseURL, logger)
	h := handler.NewURLHandler(svc, logger)

	router := handler.SetupRoutes(h)

	logger.Sugar().Infof("Запускаем сервер по адресу %s", cfg.ServerAddress.String())

	if err := http.ListenAndServe(cfg.ServerAddress.String(), router); err != nil {
		logger.Sugar().Fatalf("Ошибка при запуске: %v", err)
	}
}
