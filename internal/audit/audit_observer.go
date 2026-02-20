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

// AuditObserver получает события аудита от Publisher.
type AuditObserver interface {
	// Notify доставляет событие аудита наблюдателю.
	Notify(event model.AuditEvent) error
	// GetID возвращает уникальный идентификатор типа наблюдателя.
	GetID() string
}

// FileObserver — AuditObserver, дописывающий события в файл в формате JSON-lines.
type FileObserver struct {
	file *os.File
	mu   sync.Mutex
}

// NewFileObserver создаёт FileObserver, записывающий в указанный файл.
func NewFileObserver(file *os.File) *FileObserver {
	return &FileObserver{
		file: file,
	}
}

func (f *FileObserver) Notify(event model.AuditEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	data = append(data, '\n')

	f.mu.Lock()
	if _, err = f.file.Write(data); err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}
	defer f.mu.Unlock()

	return nil
}

func (f *FileObserver) GetID() string {
	return "file"
}

// HTTPObserver — AuditObserver, отправляющий события через HTTP POST в формате JSON.
type HTTPObserver struct {
	url string
}

// NewHTTPObserver создаёт HTTPObserver, отправляющий события на указанный URL.
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
