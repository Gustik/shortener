package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/Gustik/shortener/internal/model"
)

const pgDuplicateErrorCode = "23505"

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
		// Если INSERT был пропущен из-за конфликта по original_url, RETURNING ничего не вернёт
		if err == pgx.ErrNoRows {
			// Получаем существующую запись
			existsURL, err := r.getByOriginalURL(ctx, originalURL)
			if err != nil {
				return nil, err
			}

			return existsURL, ErrURLConflict
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgDuplicateErrorCode {
			// Это unique violation - возможно по short_url
			if strings.Contains(pgErr.ConstraintName, "short_url") {
				return nil, ErrShortURLConflict
			}
		}

		return nil, fmt.Errorf("ошибка сохранения URL: %w", err)
	}

	return &record, nil
}

func (r SQLURLRepository) SaveBatch(ctx context.Context, records []model.URLRecord) ([]model.URLRecord, error) {
	tx, err := r.conn.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer tx.Rollback(ctx)

	result := make([]model.URLRecord, len(records))

	for i, record := range records {
		query := `
			INSERT INTO urls (short_url, original_url)
			VALUES ($1, $2)
			ON CONFLICT (original_url) DO UPDATE SET original_url = EXCLUDED.original_url
			RETURNING id, short_url, original_url
		`

		err := tx.QueryRow(ctx, query, record.ShortURL, record.OriginalURL).Scan(
			&result[i].UUID,
			&result[i].ShortURL,
			&result[i].OriginalURL,
		)

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgDuplicateErrorCode {
			// Это unique violation - возможно по short_url
			if strings.Contains(pgErr.ConstraintName, "short_url") {
				return nil, ErrShortURLConflict
			}
		}

		if err != nil {
			return nil, fmt.Errorf("ошибка сохранения URL %s: %w", record.OriginalURL, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("ошибка коммита транзакции: %w", err)
	}

	return result, nil
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

func (r SQLURLRepository) getByOriginalURL(ctx context.Context, originalURL string) (*model.URLRecord, error) {
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

func (r SQLURLRepository) Ping(ctx context.Context) error {
	return r.conn.Ping(ctx)
}
