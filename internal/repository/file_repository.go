package repository

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"sync"

	"github.com/google/uuid"

	"github.com/Gustik/shortener/internal/model"
)

type FileURLRepository struct {
	mu       sync.RWMutex
	filePath string
}

func NewFileURLRepository(filePath string) (*FileURLRepository, error) {
	repo := &FileURLRepository{
		filePath: filePath,
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		file, err := os.Create(filePath)
		if err != nil {
			return nil, err
		}
		file.Close()
	}

	return repo, nil
}

func (r *FileURLRepository) Save(ctx context.Context, shortURL, originalURL string) (*model.URLRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	exists, err := r.checkExists(shortURL)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrURLExists
	}

	record := &model.URLRecord{
		UUID:        uuid.New(),
		ShortURL:    shortURL,
		OriginalURL: originalURL,
	}

	if err := r.appendRecord(record); err != nil {
		return nil, err
	}

	return record, nil
}

func (r *FileURLRepository) GetByShortURL(ctx context.Context, shortURL string) (*model.URLRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.findByShortURL(shortURL)
}

func (r *FileURLRepository) appendRecord(record *model.URLRecord) error {
	file, err := os.OpenFile(r.filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	data = append(data, '\n')

	_, err = file.Write(data)
	return err
}

func (r *FileURLRepository) findByShortURL(shortURL string) (*model.URLRecord, error) {
	file, err := os.Open(r.filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var record model.URLRecord
		if err := json.Unmarshal(line, &record); err != nil {
			continue // пропускаем поврежденные строки
		}

		if record.ShortURL == shortURL {
			return &record, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return nil, ErrURLNotFound
}

func (r *FileURLRepository) checkExists(shortURL string) (bool, error) {
	_, err := r.findByShortURL(shortURL)
	if err == ErrURLNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
