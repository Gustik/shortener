package config

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigPriority(t *testing.T) {
	tests := []struct {
		name          string
		envVars       map[string]string
		args          []string
		wantAddr      string
		wantBaseURL   string
		wantFilePath  string
		wantLogLevel  string
	}{
		{
			name:         "default values when no env and no flags",
			envVars:      map[string]string{},
			args:         []string{},
			wantAddr:     "localhost:8080",
			wantBaseURL:  "http://localhost:8080",
			wantFilePath: "db.json",
			wantLogLevel: "info",
		},
		{
			name:    "env variables override defaults",
			envVars: map[string]string{
				"SERVER_ADDRESS":    "env.host:9090",
				"BASE_URL":          "http://env.url",
				"FILE_STORAGE_PATH": "env.json",
				"LOG_LEVEL":         "debug",
			},
			args:         []string{},
			wantAddr:     "env.host:9090",
			wantBaseURL:  "http://env.url",
			wantFilePath: "env.json",
			wantLogLevel: "debug",
		},
		{
			name:    "flags override defaults",
			envVars: map[string]string{},
			args: []string{
				"-a", "flag.host:7070",
				"-b", "http://flag.url",
				"-f", "flag.json",
				"-l", "warn",
			},
			wantAddr:     "flag.host:7070",
			wantBaseURL:  "http://flag.url",
			wantFilePath: "flag.json",
			wantLogLevel: "warn",
		},
		{
			name: "env variables override flags (priority check)",
			envVars: map[string]string{
				"SERVER_ADDRESS":    "env.host:9090",
				"BASE_URL":          "http://env.url",
				"FILE_STORAGE_PATH": "env.json",
				"LOG_LEVEL":         "debug",
			},
			args: []string{
				"-a", "flag.host:7070",
				"-b", "http://flag.url",
				"-f", "flag.json",
				"-l", "warn",
			},
			wantAddr:     "env.host:9090",
			wantBaseURL:  "http://env.url",
			wantFilePath: "env.json",
			wantLogLevel: "debug",
		},
		{
			name: "mixed env and flags",
			envVars: map[string]string{
				"SERVER_ADDRESS": "env.host:9090",
				"LOG_LEVEL":      "debug",
			},
			args: []string{
				"-b", "http://flag.url",
				"-f", "flag.json",
			},
			wantAddr:     "env.host:9090",
			wantBaseURL:  "http://flag.url",
			wantFilePath: "flag.json",
			wantLogLevel: "debug",
		},
		{
			name: "partial env overrides",
			envVars: map[string]string{
				"BASE_URL": "http://env.url",
			},
			args: []string{
				"-a", "flag.host:7070",
			},
			wantAddr:     "flag.host:7070",
			wantBaseURL:  "http://env.url",
			wantFilePath: "db.json",
			wantLogLevel: "info",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Сохраняем текущее состояние
			oldArgs := os.Args
			oldEnv := make(map[string]string)
			envKeys := []string{"SERVER_ADDRESS", "BASE_URL", "FILE_STORAGE_PATH", "LOG_LEVEL"}

			// Сохраняем старые значения env
			for _, key := range envKeys {
				if val, ok := os.LookupEnv(key); ok {
					oldEnv[key] = val
				}
			}

			// Очищаем все env переменные
			for _, key := range envKeys {
				os.Unsetenv(key)
			}

			// Устанавливаем новые env переменные
			for key, val := range tt.envVars {
				os.Setenv(key, val)
			}

			// Сбрасываем флаги
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			// Устанавливаем аргументы командной строки
			os.Args = append([]string{"cmd"}, tt.args...)

			// Загружаем конфигурацию
			cfg := Load()

			// Проверяем результаты
			assert.Equal(t, tt.wantAddr, cfg.ServerAddress.String(), "ServerAddress mismatch")
			assert.Equal(t, tt.wantBaseURL, cfg.BaseURL, "BaseURL mismatch")
			assert.Equal(t, tt.wantFilePath, cfg.FileStoragePath, "FileStoragePath mismatch")
			assert.Equal(t, tt.wantLogLevel, cfg.LogLevel, "LogLevel mismatch")

			// Восстанавливаем состояние
			os.Args = oldArgs

			// Очищаем тестовые env переменные
			for _, key := range envKeys {
				os.Unsetenv(key)
			}

			// Восстанавливаем старые env переменные
			for key, val := range oldEnv {
				os.Setenv(key, val)
			}

			// Сбрасываем флаги
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		})
	}
}

func TestGetConfigValue(t *testing.T) {
	tests := []struct {
		name         string
		envKey       string
		envValue     string
		setEnv       bool
		flagValue    string
		defaultValue string
		want         string
	}{
		{
			name:         "env has priority over flag",
			envKey:       "TEST_VAR",
			envValue:     "env-value",
			setEnv:       true,
			flagValue:    "flag-value",
			defaultValue: "default-value",
			want:         "env-value",
		},
		{
			name:         "flag has priority over default",
			envKey:       "TEST_VAR",
			setEnv:       false,
			flagValue:    "flag-value",
			defaultValue: "default-value",
			want:         "flag-value",
		},
		{
			name:         "default when no env and no flag",
			envKey:       "TEST_VAR",
			setEnv:       false,
			flagValue:    "",
			defaultValue: "default-value",
			want:         "default-value",
		},
		{
			name:         "empty env value still overrides",
			envKey:       "TEST_VAR",
			envValue:     "",
			setEnv:       true,
			flagValue:    "flag-value",
			defaultValue: "default-value",
			want:         "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Очищаем переменную окружения
			os.Unsetenv(tt.envKey)

			if tt.setEnv {
				os.Setenv(tt.envKey, tt.envValue)
				defer os.Unsetenv(tt.envKey)
			}

			got := getConfigValue(tt.envKey, tt.flagValue, tt.defaultValue)
			assert.Equal(t, tt.want, got, "getConfigValue() result mismatch")
		})
	}
}

func TestNetAddrSetAndString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantStr string
		wantErr bool
	}{
		{
			name:    "valid address",
			input:   "localhost:8080",
			wantStr: "localhost:8080",
			wantErr: false,
		},
		{
			name:    "valid IP address",
			input:   "127.0.0.1:3000",
			wantStr: "127.0.0.1:3000",
			wantErr: false,
		},
		{
			name:    "invalid format - no port",
			input:   "localhost",
			wantErr: true,
		},
		{
			name:    "invalid format - non-numeric port",
			input:   "localhost:abc",
			wantErr: true,
		},
		{
			name:    "invalid format - too many colons",
			input:   "localhost:8080:9090",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			addr := &NetAddr{}
			err := addr.Set(tt.input)

			if tt.wantErr {
				assert.Error(t, err, "NetAddr.Set() should return error")
			} else {
				require.NoError(t, err, "NetAddr.Set() should not return error")
				assert.Equal(t, tt.wantStr, addr.String(), "NetAddr.String() mismatch")
			}
		})
	}
}
