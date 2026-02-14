package audit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/Gustik/shortener/internal/model"
)

type AuditObserver interface {
	Notify(event model.AuditEvent) error
	GetID() string
}

type FileObserver struct {
	file *os.File
	mu   sync.Mutex
}

func NewFileObserver(file *os.File) *FileObserver {
	return &FileObserver{
		file: file,
	}
}

func (f *FileObserver) Notify(event model.AuditEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	data = append(data, '\n')

	if _, err = f.file.Write(data); err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}

	return nil
}

func (f *FileObserver) GetID() string {
	return "file"
}

type HTTPObserver struct {
	url string
}

func NewHTTPObserver(url string) *HTTPObserver {
	return &HTTPObserver{url: url}
}

func (h *HTTPObserver) Notify(event model.AuditEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	resp, err := http.Post(h.url, "application/json", bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to send audit event: %w", err)
	}
	defer resp.Body.Close()

	return nil
}

func (h *HTTPObserver) GetID() string {
	return "http"
}
