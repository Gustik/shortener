package config

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/ilyakaznacheev/cleanenv"
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
	defaultDBMaxConns    = 25
	defaultDBMinConns    = 2
)

// NetAddr — сетевой адрес, состоящий из хоста и порта.
// Реализует flag.Value и encoding.TextUnmarshaler.
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

// UnmarshalText реализует encoding.TextUnmarshaler — используется cleanenv и json.Unmarshal.
func (n *NetAddr) UnmarshalText(text []byte) error {
	return n.Set(string(text))
}

// MarshalText реализует encoding.TextMarshaler.
func (n *NetAddr) MarshalText() ([]byte, error) {
	return []byte(n.String()), nil
}

// Config хранит конфигурацию приложения.
// Теги env задают имена переменных окружения (читает cleanenv).
// Теги json задают ключи файла конфигурации.
type Config struct {
	ServerAddress   string `env:"SERVER_ADDRESS"    json:"server_address"`
	BaseURL         string `env:"BASE_URL"          json:"base_url"`
	LogLevel        string `env:"LOG_LEVEL"         json:"log_level"`
	FileStoragePath string `env:"FILE_STORAGE_PATH" json:"file_storage_path"`
	DatabaseDSN     string `env:"DATABASE_DSN"      json:"database_dsn"`
	JWTSecret       string `env:"JWT_SECRET"`
	AuditFile       string `env:"AUDIT_FILE"        json:"audit_file"`
	AuditURL        string `env:"AUDIT_URL"         json:"audit_url"`
	EnableHTTPS     bool   `env:"ENABLE_HTTPS"      json:"enable_https"`
	TrustedSubnet   string `env:"TRUSTED_SUBNET"    json:"trusted_subnet"`
	PprofEnabled    bool   `env:"PPROF_ENABLED"`
	DBMaxConns      int    `env:"DB_MAX_CONNECTIONS"`
	DBMinConns      int    `env:"DB_MIN_CONNECTIONS"`
	StorageType     string `env:"-"` // вычисляется, не читается из env/файла
}

// Flags хранит значения флагов командной строки.
type Flags struct {
	ServerAddr      string
	BaseURL         string
	LogLevel        string
	FileStoragePath string
	DatabaseDSN     string
	AuditFile       string
	AuditURL        string
	EnableHTTPS     bool
	TrustedSubnet   string
	ConfigFile      string
	DBMaxConns      int
	DBMinConns      int
}

// Load создаёт Config, объединяя источники в порядке приоритета:
// переменные окружения > флаги CLI > файл конфигурации > значения по умолчанию.
func Load() (*Config, error) {
	flags := parseFlags()

	configPath := flags.ConfigFile
	if envPath, ok := os.LookupEnv("CONFIG"); ok && envPath != "" {
		configPath = envPath
	}

	cfg := &Config{
		ServerAddress: defaultServerAddress,
		BaseURL:       defaultBaseURL,
		LogLevel:      defaultLogLevel,
		JWTSecret:     defaultJWTSecret,
		DBMaxConns:    defaultDBMaxConns,
		DBMinConns:    defaultDBMinConns,
	}

	// Файл конфигурации — наименьший приоритет после дефолтов.
	if configPath != "" {
		if err := cleanenv.ReadConfig(configPath, cfg); err != nil {
			return nil, fmt.Errorf("ошибка чтения конфига %s: %w", configPath, err)
		}
	}

	// Флаги перекрывают файл.
	applyFlags(cfg, flags)

	// Переменные окружения перекрывают флаги.
	if err := cleanenv.ReadEnv(cfg); err != nil {
		return nil, fmt.Errorf("ошибка чтения ENV: %w", err)
	}

	if cfg.DatabaseDSN != "" {
		cfg.StorageType = StorageSQL
	} else if cfg.FileStoragePath != "" {
		cfg.StorageType = StorageFile
	} else {
		cfg.StorageType = StorageMem
	}

	printConfigInfo(cfg)
	return cfg, nil
}

func applyFlags(cfg *Config, flags *Flags) {
	if flags.ServerAddr != "" {
		cfg.ServerAddress = flags.ServerAddr
	}
	if flags.BaseURL != "" {
		cfg.BaseURL = flags.BaseURL
	}
	if flags.LogLevel != "" {
		cfg.LogLevel = flags.LogLevel
	}
	if flags.FileStoragePath != "" {
		cfg.FileStoragePath = flags.FileStoragePath
	}
	if flags.DatabaseDSN != "" {
		cfg.DatabaseDSN = flags.DatabaseDSN
	}
	if flags.AuditFile != "" {
		cfg.AuditFile = flags.AuditFile
	}
	if flags.AuditURL != "" {
		cfg.AuditURL = flags.AuditURL
	}
	if flags.EnableHTTPS {
		cfg.EnableHTTPS = true
	}
	if flags.TrustedSubnet != "" {
		cfg.TrustedSubnet = flags.TrustedSubnet
	}
	if flags.DBMaxConns != 0 {
		cfg.DBMaxConns = flags.DBMaxConns
	}
	if flags.DBMinConns != 0 {
		cfg.DBMinConns = flags.DBMinConns
	}
}

func parseFlags() *Flags {
	f := &Flags{}
	flag.StringVar(&f.ServerAddr, "a", "", "адрес и порт сервера в формате host:port")
	flag.StringVar(&f.BaseURL, "b", "", "базовый URL для сокращенных ссылок")
	flag.StringVar(&f.FileStoragePath, "f", "", "путь файла данных")
	flag.StringVar(&f.DatabaseDSN, "d", "", "DSN подключения к бд")
	flag.StringVar(&f.LogLevel, "l", "", "уровень логирования")
	flag.BoolVar(&f.EnableHTTPS, "s", false, "включить HTTPS")
	flag.StringVar(&f.TrustedSubnet, "t", "", "доверенная подсеть в формате CIDR")
	flag.StringVar(&f.AuditFile, "audit-file", "", "путь файла лога аудита")
	flag.StringVar(&f.AuditURL, "audit-url", "", "URL аудита")
	flag.StringVar(&f.ConfigFile, "c", "", "путь к файлу конфигурации (JSON)")
	flag.StringVar(&f.ConfigFile, "config", "", "путь к файлу конфигурации (JSON)")
	flag.IntVar(&f.DBMaxConns, "db-max-conns", 0, "максимальное количество соединений в пуле БД")
	flag.IntVar(&f.DBMinConns, "db-min-conns", 0, "минимальное количество соединений в пуле БД")
	flag.Parse()
	return f
}

func printConfigInfo(cfg *Config) {
	log.Println("Конфигурация загружена")
	log.Println("---")
	log.Println("addr:", cfg.ServerAddress)
	log.Println("baseURL:", cfg.BaseURL)
	log.Println("logLevel:", cfg.LogLevel)
	log.Println("fileStoragePath:", cfg.FileStoragePath)
	log.Println("databaseDSN:", cfg.DatabaseDSN)
	log.Println("storageType:", cfg.StorageType)
	log.Println("---")
}
