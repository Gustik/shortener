package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/Gustik/shortener/internal/model"
)

type SQLURLRepository struct {
	conn *pgx.Conn
}

func NewSQLRepository(conn *pgx.Conn) (*SQLURLRepository, error) {
	return &SQLURLRepository{
		conn: conn,
	}, nil
}

func (r SQLURLRepository) Save(ctx context.Context, shortURL, originalURL string) (*model.URLRecord, error) {
	query := `
        INSERT INTO urls (short_url, original_url) 
        VALUES ($1, $2)
        ON CONFLICT (original_url) DO NOTHING
        RETURNING id, short_url, original_url
    `

	var record model.URLRecord
	err := r.conn.QueryRow(ctx, query, shortURL, originalURL).Scan(
		&record.UUID,
		&record.ShortURL,
		&record.OriginalURL,
	)

	if err != nil {
		// Если INSERT был пропущен из-за конфликта, RETURNING ничего не вернёт
		if err == pgx.ErrNoRows {
			// Получаем существующую запись
			existsURL, err := r.GetByOriginalURL(ctx, originalURL)
			if err != nil {
				return nil, err
			}

			return existsURL, ErrURLExists
		}
		return nil, fmt.Errorf("ошибка сохранения URL: %w", err)
	}

	return &record, nil
}

func (r SQLURLRepository) GetByShortURL(ctx context.Context, shortURL string) (*model.URLRecord, error) {
	query := `SELECT id, short_url, original_url FROM urls WHERE short_url = $1`

	var record model.URLRecord
	err := r.conn.QueryRow(ctx, query, shortURL).Scan(
		&record.UUID,
		&record.ShortURL,
		&record.OriginalURL,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrURLNotFound
		}
		return nil, fmt.Errorf("ошибка получения URL: %w", err)
	}

	return &record, nil
}

func (r SQLURLRepository) GetByOriginalURL(ctx context.Context, originalURL string) (*model.URLRecord, error) {
	query := `SELECT id, short_url, original_url FROM urls WHERE original_url = $1`

	var record model.URLRecord
	err := r.conn.QueryRow(ctx, query, originalURL).Scan(
		&record.UUID,
		&record.ShortURL,
		&record.OriginalURL,
	)

	if err != nil {
		return nil, fmt.Errorf("ошибка получения URL: %w", err)
	}

	return &record, nil
}
