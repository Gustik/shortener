package main

import (
	"os"
	"testing"

	"go.uber.org/zap"

	"github.com/Gustik/shortener/internal/config"
)

func TestNewRepositories_Memory(t *testing.T) {
	cfg := &config.Config{StorageType: config.StorageMem}

	repos, err := NewRepositories(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer repos.Close()

	if repos.Repo == nil {
		t.Fatal("expected non-nil repository")
	}
}

func TestNewRepositories_File(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "repo-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	cfg := &config.Config{
		StorageType:     config.StorageFile,
		FileStoragePath: f.Name(),
	}

	repos, err := NewRepositories(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer repos.Close()

	if repos.Repo == nil {
		t.Fatal("expected non-nil repository")
	}
}

func TestNewRepositories_File_InvalidPath(t *testing.T) {
	cfg := &config.Config{
		StorageType:     config.StorageFile,
		FileStoragePath: "/nonexistent/path/that/cannot/be/created/repo.jsonl",
	}

	_, err := NewRepositories(cfg, zap.NewNop())
	if err == nil {
		t.Fatal("expected error for invalid path")
	}
}
