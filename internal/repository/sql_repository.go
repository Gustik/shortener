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

func (r SQLURLRepository) Save(ctx context.Context, shortURL, originalURL, userID string) (*model.URLRecord, error) {
	query := `
        INSERT INTO urls (short_url, original_url, user_id)
        VALUES ($1, $2, $3)
        ON CONFLICT (original_url) DO NOTHING
        RETURNING id, short_url, original_url, user_id, is_deleted
    `

	var record model.URLRecord
	err := r.conn.QueryRow(ctx, query, shortURL, originalURL, userID).Scan(
		&record.UUID,
		&record.ShortURL,
		&record.OriginalURL,
		&record.UserID,
		&record.IsDeleted,
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
			INSERT INTO urls (short_url, original_url, user_id)
			VALUES ($1, $2, $3)
			ON CONFLICT (original_url) DO UPDATE SET original_url = EXCLUDED.original_url
			RETURNING id, short_url, original_url, user_id, is_deleted
		`

		err := tx.QueryRow(ctx, query, record.ShortURL, record.OriginalURL, record.UserID).Scan(
			&result[i].UUID,
			&result[i].ShortURL,
			&result[i].OriginalURL,
			&result[i].UserID,
			&result[i].IsDeleted,
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
	query := `SELECT id, short_url, original_url, user_id, is_deleted FROM urls WHERE short_url = $1`

	var record model.URLRecord
	err := r.conn.QueryRow(ctx, query, shortURL).Scan(
		&record.UUID,
		&record.ShortURL,
		&record.OriginalURL,
		&record.UserID,
		&record.IsDeleted,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrURLNotFound
		}
		return nil, fmt.Errorf("ошибка получения URL: %w", err)
	}

	if record.IsDeleted {
		return nil, ErrURLDeleted
	}

	return &record, nil
}

func (r SQLURLRepository) getByOriginalURL(ctx context.Context, originalURL string) (*model.URLRecord, error) {
	query := `SELECT id, short_url, original_url, user_id, is_deleted FROM urls WHERE original_url = $1`

	var record model.URLRecord
	err := r.conn.QueryRow(ctx, query, originalURL).Scan(
		&record.UUID,
		&record.ShortURL,
		&record.OriginalURL,
		&record.UserID,
		&record.IsDeleted,
	)

	if err != nil {
		return nil, fmt.Errorf("ошибка получения URL: %w", err)
	}

	return &record, nil
}

func (r SQLURLRepository) GetByUserID(ctx context.Context, userID string) ([]model.URLRecord, error) {
	query := `SELECT id, short_url, original_url, user_id, is_deleted FROM urls WHERE user_id = $1`

	rows, err := r.conn.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения URL пользователя: %w", err)
	}
	defer rows.Close()

	var records []model.URLRecord
	for rows.Next() {
		var record model.URLRecord
		if err := rows.Scan(&record.UUID, &record.ShortURL, &record.OriginalURL, &record.UserID, &record.IsDeleted); err != nil {
			return nil, fmt.Errorf("ошибка сканирования записи: %w", err)
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка обработки строк: %w", err)
	}

	return records, nil
}

func (r SQLURLRepository) DeleteURLs(ctx context.Context, shortURLs []string, userID string) error {
	if len(shortURLs) == 0 {
		return nil
	}

	query := `
		UPDATE urls
		SET is_deleted = true
		WHERE short_url = ANY($1) AND user_id = $2
	`

	_, err := r.conn.Exec(ctx, query, shortURLs, userID)
	if err != nil {
		return fmt.Errorf("ошибка удаления URL: %w", err)
	}

	return nil
}

func (r SQLURLRepository) Ping(ctx context.Context) error {
	return r.conn.Ping(ctx)
}
