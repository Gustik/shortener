package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/Gustik/shortener/internal/config"
	"github.com/Gustik/shortener/internal/handler"
	"github.com/Gustik/shortener/internal/repository"
	"github.com/Gustik/shortener/internal/service"
	"github.com/Gustik/shortener/internal/zaplog"
)

func main() {
	cfg := config.Load()

	logger, err := zaplog.New(cfg.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка инициализации логгера: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	repo, cleanup, err := initRepository(cfg, logger)
	if err != nil {
		logger.Fatal("Ошибка инициализации репозитория", zap.Error(err))
	}
	defer cleanup()

	// Создаём контекст приложения для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc := service.NewURLService(repo, cfg.BaseURL, logger)
	h := handler.NewURLHandler(ctx, svc, logger)
	router := handler.SetupRoutes(h, cfg.JWTSecret)

	runServerWithGracefulShutdown(cancel, cfg.ServerAddress.String(), router, logger)
}

func runServerWithGracefulShutdown(cancel context.CancelFunc, addr string, handler http.Handler, logger *zap.Logger) {
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	// Канал для ожидания сигналов завершения
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Sugar().Infof("Запускаем сервер по адресу %s", addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("Ошибка при запуске сервера", zap.Error(err))
		}
	}()

	<-quit
	logger.Info("Получен сигнал завершения, начинаем graceful shutdown...")

	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Ошибка при graceful shutdown", zap.Error(err))
	}

	logger.Info("Сервер успешно остановлен")
}

func initRepository(cfg *config.Config, logger *zap.Logger) (repository.URLRepository, func(), error) {
	switch cfg.StorageType {
	case config.StorageFile:
		return initFileRepository(cfg, logger)
	case config.StorageSQL:
		return initSQLRepository(cfg, logger)
	default:
		return initMemoryRepository(logger)
	}
}

func initMemoryRepository(logger *zap.Logger) (repository.URLRepository, func(), error) {
	logger.Info("Инициализация in-memory репозитория")
	repo := repository.NewInMemoryURLRepository()
	return repo, func() {}, nil
}

func initFileRepository(cfg *config.Config, logger *zap.Logger) (repository.URLRepository, func(), error) {
	logger.Info("Инициализация file репозитория", zap.String("path", cfg.FileStoragePath))

	file, err := os.OpenFile(cfg.FileStoragePath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка открытия файла репозитория: %w", err)
	}

	repo, err := repository.NewFileURLRepository(file)
	if err != nil {
		file.Close()
		return nil, nil, fmt.Errorf("ошибка инициализации file репозитория: %w", err)
	}

	cleanup := func() {
		if err := file.Close(); err != nil {
			logger.Error("Ошибка закрытия файла репозитория", zap.Error(err))
		}
	}

	return repo, cleanup, nil
}

func initSQLRepository(cfg *config.Config, logger *zap.Logger) (repository.URLRepository, func(), error) {
	logger.Info("Запуск миграций БД")
	if err := runMigrations(cfg.DatabaseDSN); err != nil {
		return nil, nil, fmt.Errorf("ошибка применения миграций: %w", err)
	}
	logger.Info("Миграции успешно применены")

	logger.Info("Подключение к PostgreSQL")
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseDSN)
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка подключения к БД: %w", err)
	}

	repo, err := repository.NewSQLRepository(pool)
	if err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("ошибка инициализации SQL репозитория: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := repo.Ping(ctx); err != nil {
		pool.Close()
		return nil, nil, fmt.Errorf("ошибка проверки подключения к БД: %w", err)
	}
	logger.Info("Успешное подключение к PostgreSQL")

	cleanup := func() {
		logger.Info("Закрываю соединение с posgtres")
		pool.Close()
	}

	return repo, cleanup, nil
}

func runMigrations(databaseDSN string) error {
	m, err := migrate.New(
		"file://migrations",
		databaseDSN,
	)
	if err != nil {
		return fmt.Errorf("ошибка создания migrate instance: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("ошибка применения миграций: %w", err)
	}

	return nil
}
