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
		name         string
		envVars      map[string]string
		args         []string
		wantAddr     string
		wantBaseURL  string
		wantLogLevel string
	}{
		{
			name:         "default values when no env and no flags",
			envVars:      map[string]string{},
			args:         []string{},
			wantAddr:     "localhost:8080",
			wantBaseURL:  "http://localhost:8080",
			wantLogLevel: "info",
		},
		{
			name: "env variables override defaults",
			envVars: map[string]string{
				"SERVER_ADDRESS": "env.host:9090",
				"BASE_URL":       "http://env.url",
				"LOG_LEVEL":      "debug",
			},
			args:         []string{},
			wantAddr:     "env.host:9090",
			wantBaseURL:  "http://env.url",
			wantLogLevel: "debug",
		},
		{
			name:    "flags override defaults",
			envVars: map[string]string{},
			args: []string{
				"-a", "flag.host:7070",
				"-b", "http://flag.url",
				"-l", "warn",
			},
			wantAddr:     "flag.host:7070",
			wantBaseURL:  "http://flag.url",
			wantLogLevel: "warn",
		},
		{
			name: "env variables override flags (priority check)",
			envVars: map[string]string{
				"BASE_URL":  "http://env.url",
				"LOG_LEVEL": "debug",
			},
			args: []string{
				"-b", "http://flag.url",
				"-l", "warn",
			},
			wantAddr:     "localhost:8080",
			wantBaseURL:  "http://env.url",
			wantLogLevel: "debug",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldArgs := os.Args
			oldEnv := make(map[string]string)
			envKeys := []string{"SERVER_ADDRESS", "BASE_URL", "FILE_STORAGE_PATH", "DATABASE_DSN", "LOG_LEVEL"}

			for _, key := range envKeys {
				if val, ok := os.LookupEnv(key); ok {
					oldEnv[key] = val
				}
				os.Unsetenv(key)
			}

			for key, val := range tt.envVars {
				os.Setenv(key, val)
			}

			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			os.Args = append([]string{"cmd"}, tt.args...)

			cfg := Load()

			assert.Equal(t, tt.wantAddr, cfg.ServerAddress.String())
			assert.Equal(t, tt.wantBaseURL, cfg.BaseURL)
			assert.Equal(t, tt.wantLogLevel, cfg.LogLevel)

			os.Args = oldArgs
			for _, key := range envKeys {
				os.Unsetenv(key)
			}
			for key, val := range oldEnv {
				os.Setenv(key, val)
			}
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
		})
	}
}

func TestStorageTypePriority(t *testing.T) {
	tests := []struct {
		name            string
		envVars         map[string]string
		args            []string
		wantStorageType string
		wantDBDSN       string
		wantFilePath    string
	}{
		{
			name:            "memory storage by default",
			envVars:         map[string]string{},
			args:            []string{},
			wantStorageType: StorageMem,
			wantDBDSN:       "",
			wantFilePath:    "",
		},
		{
			name: "file storage when only file path provided",
			envVars: map[string]string{
				"FILE_STORAGE_PATH": "/tmp/data.json",
			},
			args:            []string{},
			wantStorageType: StorageFile,
			wantFilePath:    "/tmp/data.json",
			wantDBDSN:       "",
		},
		{
			name: "sql storage when database DSN provided",
			envVars: map[string]string{
				"DATABASE_DSN": "postgres://user:pass@localhost/db",
			},
			args:            []string{},
			wantStorageType: StorageSQL,
			wantDBDSN:       "postgres://user:pass@localhost/db",
			wantFilePath:    "",
		},
		{
			name: "sql storage has priority over file storage",
			envVars: map[string]string{
				"DATABASE_DSN":      "postgres://user:pass@localhost/db",
				"FILE_STORAGE_PATH": "/tmp/data.json",
			},
			args:            []string{},
			wantStorageType: StorageSQL,
			wantDBDSN:       "postgres://user:pass@localhost/db",
			wantFilePath:    "/tmp/data.json",
		},
		{
			name:    "flags work for database DSN",
			envVars: map[string]string{},
			args: []string{
				"-d", "postgres://flag:pass@localhost/db",
			},
			wantStorageType: StorageSQL,
			wantDBDSN:       "postgres://flag:pass@localhost/db",
		},
		{
			name:    "flags work for file path",
			envVars: map[string]string{},
			args: []string{
				"-f", "/flag/path.json",
			},
			wantStorageType: StorageFile,
			wantFilePath:    "/flag/path.json",
		},
		{
			name: "empty DATABASE_DSN falls back to file storage",
			envVars: map[string]string{
				"DATABASE_DSN":      "",
				"FILE_STORAGE_PATH": "/tmp/data.json",
			},
			args:            []string{},
			wantStorageType: StorageFile,
			wantFilePath:    "/tmp/data.json",
		},
		{
			name: "empty DATABASE_DSN and FILE_STORAGE_PATH falls back to memory",
			envVars: map[string]string{
				"DATABASE_DSN":      "",
				"FILE_STORAGE_PATH": "",
			},
			args:            []string{},
			wantStorageType: StorageMem,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldArgs := os.Args
			oldEnv := make(map[string]string)
			envKeys := []string{"SERVER_ADDRESS", "BASE_URL", "FILE_STORAGE_PATH", "DATABASE_DSN", "LOG_LEVEL"}

			for _, key := range envKeys {
				if val, ok := os.LookupEnv(key); ok {
					oldEnv[key] = val
				}
				os.Unsetenv(key)
			}

			for key, val := range tt.envVars {
				os.Setenv(key, val)
			}

			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
			os.Args = append([]string{"cmd"}, tt.args...)

			cfg := Load()

			assert.Equal(t, tt.wantStorageType, cfg.StorageType, "StorageType mismatch")
			assert.Equal(t, tt.wantDBDSN, cfg.DatabaseDSN, "DatabaseDSN mismatch")
			assert.Equal(t, tt.wantFilePath, cfg.FileStoragePath, "FileStoragePath mismatch")

			os.Args = oldArgs
			for _, key := range envKeys {
				os.Unsetenv(key)
			}
			for key, val := range oldEnv {
				os.Setenv(key, val)
			}
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
			os.Unsetenv(tt.envKey)

			if tt.setEnv {
				os.Setenv(tt.envKey, tt.envValue)
				defer os.Unsetenv(tt.envKey)
			}

			got := getConfigValue(tt.envKey, tt.flagValue, tt.defaultValue)
			assert.Equal(t, tt.want, got)
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
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantStr, addr.String())
			}
		})
	}
}
