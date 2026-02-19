package config

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// Константы типа хранилища определяют, какая реализация URLRepository используется.
const (
	StorageMem  string = "mem"  // хранение в памяти
	StorageFile string = "file" // хранение в файле (JSON-lines)
	StorageSQL  string = "sql"  // хранение в PostgreSQL
)

const (
	defaultServerAddress = "localhost:8080"
	defaultBaseURL       = "http://localhost:8080"
	defaultLogLevel      = "info"
	defaultJWTSecret     = "default-secret-key-change-in-production"
)

// NetAddr — сетевой адрес, состоящий из хоста и порта.
// Реализует интерфейс flag.Value для использования с flag.Var.
type NetAddr struct {
	Host string
	Port int
}

// String возвращает адрес в формате "host:port".
func (n *NetAddr) String() string {
	if n.Host == "" && n.Port == 0 {
		return ""
	}

	return fmt.Sprintf("%s:%d", n.Host, n.Port)
}

// Set разбирает строку "host:port" и заполняет поля NetAddr.
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

// Config хранит конфигурацию приложения, собранную из
// переменных окружения, флагов командной строки и значений по умолчанию.
type Config struct {
	ServerAddress   NetAddr
	BaseURL         string
	LogLevel        string
	FileStoragePath string
	DatabaseDSN     string
	StorageType     string
	JWTSecret       string
	AuditFile       string
	AuditURL        string
	PprofEnabled    bool
}

// Flags хранит значения, полученные из флагов командной строки.
type Flags struct {
	ServerAddr      string
	BaseURL         string
	LogLevel        string
	FileStoragePath string
	DatabaseDSN     string
	JWTSecret       string
	AuditFile       string
	AuditURL        string
}

// Load создаёт Config, объединяя переменные окружения, флаги командной
// строки и значения по умолчанию (в указанном порядке приоритета).
func Load() *Config {
	cfg := &Config{}

	cfg.ServerAddress.Set(defaultServerAddress)
	cfg.BaseURL = defaultBaseURL
	cfg.LogLevel = defaultLogLevel
	cfg.StorageType = StorageMem

	flags := parseFlags()

	if envServerAddr, ok := os.LookupEnv("SERVER_ADDRESS"); ok {
		cfg.ServerAddress.Set(envServerAddr)
	} else if flags.ServerAddr != "" {
		cfg.ServerAddress.Set(flags.ServerAddr)
	}

	cfg.BaseURL = getConfigValue("BASE_URL", flags.BaseURL, defaultBaseURL)
	cfg.LogLevel = getConfigValue("LOG_LEVEL", flags.LogLevel, defaultLogLevel)
	cfg.JWTSecret = getConfigValue("JWT_SECRET", flags.JWTSecret, defaultJWTSecret)

	cfg.FileStoragePath = getConfigValue("FILE_STORAGE_PATH", flags.FileStoragePath, "")
	cfg.DatabaseDSN = getConfigValue("DATABASE_DSN", flags.DatabaseDSN, "")
	cfg.AuditFile = getConfigValue("AUDIT_FILE", flags.AuditFile, "")
	cfg.AuditURL = getConfigValue("AUDIT_URL", flags.AuditURL, "")

	if v, ok := os.LookupEnv("PPROF_ENABLED"); ok && v == "true" {
		cfg.PprofEnabled = true
	}

	if cfg.DatabaseDSN != "" {
		cfg.StorageType = StorageSQL
	} else if cfg.FileStoragePath != "" {
		cfg.StorageType = StorageFile
	}

	printConfigInfo(cfg)

	return cfg
}

func parseFlags() *Flags {
	f := &Flags{}
	flag.StringVar(&f.ServerAddr, "a", "", "адрес и порт сервера в формате host:port")
	flag.StringVar(&f.BaseURL, "b", "", "базовый URL для сокращенных ссылок")
	flag.StringVar(&f.FileStoragePath, "f", "", "путь файла данных")
	flag.StringVar(&f.DatabaseDSN, "d", "", "DSN подключения к бд")
	flag.StringVar(&f.LogLevel, "l", "", "уровень логирования")
	flag.StringVar(&f.JWTSecret, "s", "", "секретный ключ для JWT")
	flag.StringVar(&f.AuditFile, "audit-file", "", "путь файла лога аудита")
	flag.StringVar(&f.AuditURL, "audit-url", "", "URL аудита")
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
	log.Println("logLevel:", cfg.LogLevel)
	log.Println("fileStoragePath:", cfg.FileStoragePath)
	log.Println("databaseDSN:", cfg.DatabaseDSN)
	log.Println("storageType:", cfg.StorageType)
	log.Println("---")
}
