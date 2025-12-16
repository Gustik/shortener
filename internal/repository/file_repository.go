package repository

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/Gustik/shortener/internal/model"
)

type FileURLRepository struct {
	InMemoryURLRepository
	file   *os.File
	writer *bufio.Writer
}

func NewFileURLRepository(file *os.File) (*FileURLRepository, error) {
	repo := &FileURLRepository{
		InMemoryURLRepository: *NewInMemoryURLRepository(),
		file:                  file,
		writer:                bufio.NewWriter(file),
	}

	// Загружаем существующие данные построчно
	if err := repo.loadFromFile(); err != nil {
		return nil, err
	}

	// Переходим в конец файла для дальнейшей записи
	_, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}

	return repo, nil
}

func (r *FileURLRepository) Save(ctx context.Context, shortURL, originalURL string) (*model.URLRecord, error) {
	record, err := r.InMemoryURLRepository.Save(ctx, shortURL, originalURL)
	if errors.Is(err, ErrURLConflict) {
		return record, err
	}

	if err != nil {
		return nil, err
	}

	if err := r.appendToFile(record); err != nil {
		return nil, err
	}

	return record, err
}

func (r *FileURLRepository) SaveBatch(ctx context.Context, records []model.URLRecord) ([]model.URLRecord, error) {
	// Используем SaveBatch из InMemoryURLRepository для обновления памяти
	result, err := r.InMemoryURLRepository.SaveBatch(ctx, records)
	if err != nil {
		return nil, err
	}

	// Записываем все новые записи в файл одним блоком
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := range result {
		// Проверяем, была ли запись реально добавлена (а не уже существовала)
		isNew := false
		for j := range records {
			if records[j].ShortURL == result[i].ShortURL && records[j].OriginalURL == result[i].OriginalURL {
				isNew = true
				break
			}
		}

		if isNew {
			data, err := json.Marshal(&result[i])
			if err != nil {
				return nil, fmt.Errorf("save url record: %w", err)
			}

			data = append(data, '\n')
			if _, err := r.writer.Write(data); err != nil {
				return nil, fmt.Errorf("save url record: %w", err)
			}
		}
	}

	if err := r.writer.Flush(); err != nil {
		return nil, fmt.Errorf("flush records: %w", err)
	}

	return result, nil
}

// Загрузка данных из файла (каждая запись на отдельной строке)
func (r *FileURLRepository) loadFromFile() error {
	scanner := bufio.NewScanner(r.file)
	for scanner.Scan() {
		var record model.URLRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return fmt.Errorf("load url records: %w", err)
		}
		r.urls = append(r.urls, record)
	}
	return scanner.Err()
}

// Дописываем одну запись в конец файла
func (r *FileURLRepository) appendToFile(record *model.URLRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("save url record: %w", err)
	}

	// Записываем JSON + перевод строки
	data = append(data, '\n')
	if _, err := r.writer.Write(data); err != nil {
		return fmt.Errorf("save url record: %w", err)
	}
	if err := r.writer.Flush(); err != nil {
		return fmt.Errorf("save url record: %w", err)
	}

	return nil
}
