package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/Gustik/shortener/internal/config"
	"github.com/Gustik/shortener/internal/repository"
)

// Repositories хранит репозиторий и его функцию освобождения ресурсов.
type Repositories struct {
	Repo    repository.URLRepository
	cleanup func()
}

// Close освобождает ресурсы репозитория.
func (r *Repositories) Close() {
	r.cleanup()
}

// NewRepositories создаёт репозиторий нужного типа в зависимости от конфигурации.
func NewRepositories(cfg *config.Config, logger *zap.Logger) (*Repositories, error) {
	switch cfg.StorageType {
	case config.StorageFile:
		return newFileRepositories(cfg, logger)
	case config.StorageSQL:
		return newSQLRepositories(cfg, logger)
	default:
		logger.Info("Инициализация in-memory репозитория")
		return newMemoryRepositories()
	}
}

func newMemoryRepositories() (*Repositories, error) {
	return &Repositories{
		Repo:    repository.NewInMemoryURLRepository(),
		cleanup: func() {},
	}, nil
}

func newFileRepositories(cfg *config.Config, logger *zap.Logger) (*Repositories, error) {
	logger.Info("Инициализация file репозитория", zap.String("path", cfg.FileStoragePath))

	file, err := os.OpenFile(cfg.FileStoragePath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия файла репозитория: %w", err)
	}

	repo, err := repository.NewFileURLRepository(file)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("ошибка инициализации file репозитория: %w", err)
	}

	return &Repositories{
		Repo: repo,
		cleanup: func() {
			if err := file.Close(); err != nil {
				logger.Error("Ошибка закрытия файла репозитория", zap.Error(err))
			}
		},
	}, nil
}

func newSQLRepositories(cfg *config.Config, logger *zap.Logger) (*Repositories, error) {
	logger.Info("Запуск миграций БД")
	if err := runMigrations(cfg.DatabaseDSN); err != nil {
		return nil, fmt.Errorf("ошибка применения миграций: %w", err)
	}
	logger.Info("Миграции успешно применены")

	logger.Info("Подключение к PostgreSQL")
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseDSN)
	if err != nil {
		return nil, fmt.Errorf("неверный DSN: %w", err)
	}
	poolCfg.MaxConns = int32(cfg.DBMaxConns)
	poolCfg.MinConns = int32(cfg.DBMinConns)
	poolCfg.MaxConnLifetime = time.Hour
	poolCfg.MaxConnIdleTime = 30 * time.Minute
	poolCfg.HealthCheckPeriod = time.Minute
	poolCfg.ConnConfig.ConnectTimeout = 10 * time.Second

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания пула: %w", err)
	}

	repo, err := repository.NewSQLRepository(pool)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("ошибка инициализации SQL репозитория: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := repo.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ошибка проверки подключения к БД: %w", err)
	}
	logger.Info("Успешное подключение к PostgreSQL")

	return &Repositories{
		Repo: repo,
		cleanup: func() {
			logger.Info("Закрываю соединение с postgres")
			pool.Close()
		},
	}, nil
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
