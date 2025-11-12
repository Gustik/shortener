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
	defaultServerAddress = "localhost:8080"
	defaultBaseURL       = "http://localhost:8080"
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
	ServerAddress NetAddr
	BaseURL       string
}

func Load() *Config {
	cfg := &Config{}

	cfg.ServerAddress.Set(defaultServerAddress)
	cfg.BaseURL = defaultBaseURL

	var serverAddrFlag string
	var baseURLFlag string
	flag.StringVar(&serverAddrFlag, "a", "", "адрес и порт сервера в формате host:port")
	flag.StringVar(&baseURLFlag, "b", "", "базовый URL для сокращенных ссылок")
	flag.Parse()

	if serverAddrFlag != "" {
		cfg.ServerAddress.Set(serverAddrFlag)
	}
	if baseURLFlag != "" {
		cfg.BaseURL = baseURLFlag
	}

	if envServerAddr := os.Getenv("SERVER_ADDRESS"); envServerAddr != "" {
		cfg.ServerAddress.Set(envServerAddr)
	}
	if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
		cfg.BaseURL = envBaseURL
	}

	log.Println("Конфигурация загружена")
	log.Println("---")
	log.Println("addr:", cfg.ServerAddress.String())
	log.Println("baseURL:", cfg.BaseURL)
	log.Println("---")

	return cfg
}
