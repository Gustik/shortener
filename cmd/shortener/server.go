package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func runServerWithGracefulShutdown(cancel context.CancelFunc, addr string, enableHTTPS bool, handler http.Handler, grpcAddr string, grpcServer *grpc.Server, logger *zap.Logger) {
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		if enableHTTPS {
			tlsCfg, err := generateSelfSignedTLS()
			if err != nil {
				logger.Fatal("Ошибка генерации TLS сертификата", zap.Error(err))
			}
			ln, err := tls.Listen("tcp", addr, tlsCfg)
			if err != nil {
				logger.Fatal("Ошибка запуска TLS listener", zap.Error(err))
			}
			logger.Sugar().Infof("Запускаем HTTPS сервер по адресу %s", addr)
			if err := server.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Fatal("Ошибка при запуске HTTPS сервера", zap.Error(err))
			}
		} else {
			logger.Sugar().Infof("Запускаем сервер по адресу %s", addr)
			if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Fatal("Ошибка при запуске сервера", zap.Error(err))
			}
		}
	}()

	go func() {
		ln, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			logger.Fatal("Ошибка запуска gRPC listener", zap.String("addr", grpcAddr), zap.Error(err))
		}
		logger.Sugar().Infof("Запускаем gRPC сервер по адресу %s", grpcAddr)
		if err := grpcServer.Serve(ln); err != nil {
			logger.Error("Ошибка при запуске gRPC сервера", zap.Error(err))
		}
	}()

	sig := <-quit
	logger.Sugar().Infof("Получен сигнал %s, начинаем graceful shutdown...", sig)

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Сначала завершаем HTTP: ждём окончания всех активных запросов.
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("Ошибка при graceful shutdown HTTP", zap.Error(err))
	}

	// Graceful shutdown gRPC сервера.
	grpcServer.GracefulStop()

	// Только после этого отменяем контекст приложения —
	// фоновые горутины (async delete и др.) успели завершить работу.
	cancel()

	logger.Info("Сервер успешно остановлен")
}

func generateSelfSignedTLS() (*tls.Config, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("ошибка генерации ключа: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Shortener"},
		},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:    []string{"localhost"},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(1, 0, 0),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания сертификата: %w", err)
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания TLS keypair: %w", err)
	}

	return &tls.Config{Certificates: []tls.Certificate{tlsCert}}, nil
}
