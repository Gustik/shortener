package config

import "os"

type Config struct {
	ServerAddress string
	BaseURL       string
}

func Load() *Config {
	serverAddr := os.Getenv("SERVER_ADDRESS")
	if serverAddr == "" {
		serverAddr = "localhost:8080"
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	return &Config{
		ServerAddress: serverAddr,
		BaseURL:       baseURL,
	}
}
