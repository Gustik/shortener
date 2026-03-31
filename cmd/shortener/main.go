package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"

	"go.uber.org/zap"

	"github.com/Gustik/shortener/internal/audit"
	"github.com/Gustik/shortener/internal/config"
	"github.com/Gustik/shortener/internal/handler"
	"github.com/Gustik/shortener/internal/service"
	"github.com/Gustik/shortener/internal/zaplog"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func printBuildInfo() {
	na := func(s string) string {
		if s == "" {
			return "N/A"
		}
		return s
	}
	fmt.Printf("Build version: %s\n", na(buildVersion))
	fmt.Printf("Build date: %s\n", na(buildDate))
	fmt.Printf("Build commit: %s\n", na(buildCommit))
}

func main() {
	printBuildInfo()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка загрузки конфигурации: %v\n", err)
		os.Exit(1)
	}

	logger, err := zaplog.New(cfg.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка инициализации логгера: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	if cfg.PprofEnabled {
		go func() {
			logger.Info("pprof server listening on :6060")
			if err := http.ListenAndServe(":6060", nil); err != nil {
				logger.Error("pprof server error", zap.Error(err))
			}
		}()
	}

	repos, err := NewRepositories(cfg, logger)
	if err != nil {
		logger.Fatal("Ошибка инициализации репозитория", zap.Error(err))
	}
	defer repos.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	auditor, auditCleanup := audit.NewPublisher(cfg, logger)
	defer auditCleanup()

	var trustedSubnet *net.IPNet
	if cfg.TrustedSubnet != "" {
		var err error
		_, trustedSubnet, err = net.ParseCIDR(cfg.TrustedSubnet)
		if err != nil {
			logger.Fatal("Неверный формат TRUSTED_SUBNET", zap.String("value", cfg.TrustedSubnet), zap.Error(err))
		}
	}

	svc := service.NewURLService(repos.Repo, cfg.BaseURL, logger)
	h := handler.NewURLHandler(ctx, svc, logger, auditor)
	router := handler.SetupRoutes(h, cfg.JWTSecret, trustedSubnet)

	runServerWithGracefulShutdown(cancel, cfg.ServerAddress, cfg.EnableHTTPS, router, logger)
}
