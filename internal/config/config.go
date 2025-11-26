package config

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

const (
	defaultServerAddress   = "localhost:8080"
	defaultBaseURL         = "http://localhost:8080"
	defaultFileStoragePath = "db.json"
	defaultLogLevel        = "info"
)

type NetAddr struct {
	Host string
	Port int
}

func (n *NetAddr) String() string {
	if n.Host == "" && n.Port == 0 {
		return ""
	}

	return fmt.Sprintf("%s:%d", n.Host, n.Port)
}

func (n *NetAddr) Set(value string) error {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return fmt.Errorf("неверный формат адреса, ожидается host:port")
	}

	port, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("порт должен быть числом: %v", err)
	}

	n.Host = parts[0]
	n.Port = port

	return nil
}

type Config struct {
	ServerAddress   NetAddr
	BaseURL         string
	FileStoragePath string
	LogLevel        string
}

func Load() *Config {
	cfg := &Config{}

	cfg.ServerAddress.Set(defaultServerAddress)
	cfg.BaseURL = defaultBaseURL
	cfg.FileStoragePath = defaultFileStoragePath
	cfg.LogLevel = defaultLogLevel

	var serverAddrFlag string
	var baseURLFlag string
	var fileStoragePathFlag string
	var logLevelFlag string
	flag.StringVar(&serverAddrFlag, "a", "", "адрес и порт сервера в формате host:port")
	flag.StringVar(&baseURLFlag, "b", "", "базовый URL для сокращенных ссылок")
	flag.StringVar(&fileStoragePathFlag, "f", "", "путь файла данных")
	flag.StringVar(&logLevelFlag, "l", "", "уровень логирования")
	flag.Parse()

	if serverAddrFlag != "" {
		cfg.ServerAddress.Set(serverAddrFlag)
	}
	if baseURLFlag != "" {
		cfg.BaseURL = baseURLFlag
	}
	if fileStoragePathFlag != "" {
		cfg.FileStoragePath = fileStoragePathFlag
	}
	if logLevelFlag != "" {
		cfg.LogLevel = logLevelFlag
	}

	if envServerAddr, ok := os.LookupEnv("SERVER_ADDRESS"); ok {
		cfg.ServerAddress.Set(envServerAddr)
	}
	if envBaseURL, ok := os.LookupEnv("BASE_URL"); ok {
		cfg.BaseURL = envBaseURL
	}
	if envLogLevel, ok := os.LookupEnv("LOG_LEVEL"); ok {
		cfg.LogLevel = envLogLevel
	}
	if fileStoragePath, ok := os.LookupEnv("FILE_STORAGE_PATH"); ok {
		cfg.FileStoragePath = fileStoragePath
	}

	log.Println("Конфигурация загружена")
	log.Println("---")
	log.Println("addr:", cfg.ServerAddress.String())
	log.Println("baseURL:", cfg.BaseURL)
	log.Println("fileStoragePath:", cfg.FileStoragePath)
	log.Println("logLevel:", cfg.LogLevel)
	log.Println("---")

	return cfg
}
