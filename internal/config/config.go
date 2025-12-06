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
	StorageMem  string = "mem"
	StorageFile string = "file"
	StorageSQL  string = "sql"
)

const (
	defaultServerAddress   = "localhost:8080"
	defaultBaseURL         = "http://localhost:8080"
	defaultStorageType     = StorageSQL
	defaultFileStoragePath = "db.json"
	defaultDatabaseDSN     = "postgres://postgres:secret@localhost:5432/shortener"
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
	StorageType     string
	DatabaseDSN     string
}

type Flags struct {
	ServerAddr      string
	BaseURL         string
	FileStoragePath string
	LogLevel        string
	StorageType     string
	DatabaseDSN     string
}

func Load() *Config {
	cfg := &Config{}

	cfg.ServerAddress.Set(defaultServerAddress)
	cfg.BaseURL = defaultBaseURL
	cfg.FileStoragePath = defaultFileStoragePath
	cfg.DatabaseDSN = defaultDatabaseDSN
	cfg.LogLevel = defaultLogLevel
	cfg.StorageType = StorageFile

	flags := parseFlags()

	if envServerAddr, ok := os.LookupEnv("SERVER_ADDRESS"); ok {
		cfg.ServerAddress.Set(envServerAddr)
	} else if flags.ServerAddr != "" {
		cfg.ServerAddress.Set(flags.ServerAddr)
	}

	cfg.BaseURL = getConfigValue("BASE_URL", flags.BaseURL, defaultBaseURL)
	cfg.FileStoragePath = getConfigValue("FILE_STORAGE_PATH", flags.FileStoragePath, defaultFileStoragePath)
	cfg.LogLevel = getConfigValue("LOG_LEVEL", flags.LogLevel, defaultLogLevel)
	cfg.StorageType = getConfigValue("STORAGE_TYPE", flags.StorageType, defaultStorageType)
	cfg.DatabaseDSN = getConfigValue("DATABASE_DSN", flags.DatabaseDSN, defaultDatabaseDSN)

	printConfigInfo(cfg)

	return cfg
}

func parseFlags() *Flags {
	f := &Flags{}
	flag.StringVar(&f.ServerAddr, "a", "", "адрес и порт сервера в формате host:port")
	flag.StringVar(&f.BaseURL, "b", "", "базовый URL для сокращенных ссылок")
	flag.StringVar(&f.FileStoragePath, "f", "", "путь файла данных")
	flag.StringVar(&f.DatabaseDSN, "d", "", "DSN подключения к бд")
	flag.StringVar(&f.StorageType, "s", "", "тип хранилища")
	flag.StringVar(&f.LogLevel, "l", "", "уровень логирования")
	flag.Parse()

	return f
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
	log.Println("databaseDSN:", cfg.DatabaseDSN)
	log.Println("storageType:", cfg.StorageType)
	log.Println("logLevel:", cfg.LogLevel)
	log.Println("---")
}
