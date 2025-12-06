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
	if errors.Is(err, ErrURLExists) {
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
