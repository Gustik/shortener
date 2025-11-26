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

	if envServerAddr, ok := os.LookupEnv("SERVER_ADDRESS"); ok {
		cfg.ServerAddress.Set(envServerAddr)
	} else if serverAddrFlag != "" {
		cfg.ServerAddress.Set(serverAddrFlag)
	}

	cfg.BaseURL = getConfigValue("BASE_URL", baseURLFlag, defaultBaseURL)
	cfg.FileStoragePath = getConfigValue("FILE_STORAGE_PATH", fileStoragePathFlag, defaultFileStoragePath)
	cfg.LogLevel = getConfigValue("LOG_LEVEL", logLevelFlag, defaultLogLevel)

	printConfigInfo(cfg)

	return cfg
}

func getConfigValue(envKey, flagValue, defaultValue string) string {
	if envValue, ok := os.LookupEnv(envKey); ok {
		return envValue
	}
	if flagValue != "" {
		return flagValue
	}
	return defaultValue
}

func printConfigInfo(cfg *Config) {
	log.Println("Конфигурация загружена")
	log.Println("---")
	log.Println("addr:", cfg.ServerAddress.String())
	log.Println("baseURL:", cfg.BaseURL)
	log.Println("fileStoragePath:", cfg.FileStoragePath)
	log.Println("logLevel:", cfg.LogLevel)
	log.Println("---")
}
