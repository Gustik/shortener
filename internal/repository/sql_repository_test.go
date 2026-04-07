package repository_test

import (
	"context"
	"fmt"
	"testing"

	pgxmock "github.com/pashagolub/pgxmock/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Gustik/shortener/internal/model"
	"github.com/Gustik/shortener/internal/repository"
)

func newMockRepo(t *testing.T) (repository.URLRepository, pgxmock.PgxPoolIface) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	repo, err := repository.NewSQLRepositoryWithDB(mock)
	require.NoError(t, err)
	return repo, mock
}

// --- Save ---

func TestSQLURLRepository_Save_New(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMockRepo(t)

	rows := pgxmock.NewRows([]string{"id", "short_url", "original_url", "user_id", "is_deleted"}).
		AddRow("550e8400-e29b-41d4-a716-446655440000", "abc12345", "https://ya.ru", "user1", false)

	mock.ExpectQuery(`INSERT INTO urls`).
		WithArgs("abc12345", "https://ya.ru", "user1").
		WillReturnRows(rows)

	rec, err := repo.Save(ctx, "abc12345", "https://ya.ru", "user1")

	require.NoError(t, err)
	assert.Equal(t, "abc12345", rec.ShortURL)
	assert.Equal(t, "https://ya.ru", rec.OriginalURL)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLURLRepository_Save_URLConflict(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMockRepo(t)

	// INSERT вернул 0 строк (ON CONFLICT DO NOTHING) → pgx.ErrNoRows
	mock.ExpectQuery(`INSERT INTO urls`).
		WithArgs("abc12345", "https://ya.ru", "user1").
		WillReturnRows(pgxmock.NewRows([]string{"id", "short_url", "original_url", "user_id", "is_deleted"}))

	// Затем getByOriginalURL
	existRows := pgxmock.NewRows([]string{"id", "short_url", "original_url", "user_id", "is_deleted"}).
		AddRow("550e8400-e29b-41d4-a716-446655440000", "existing1", "https://ya.ru", "user1", false)
	mock.ExpectQuery(`SELECT id, short_url, original_url, user_id, is_deleted FROM urls WHERE original_url`).
		WithArgs("https://ya.ru").
		WillReturnRows(existRows)

	rec, err := repo.Save(ctx, "abc12345", "https://ya.ru", "user1")

	assert.ErrorIs(t, err, repository.ErrURLConflict)
	assert.Equal(t, "existing1", rec.ShortURL)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLURLRepository_Save_InternalError(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(`INSERT INTO urls`).
		WithArgs("abc12345", "https://ya.ru", "user1").
		WillReturnError(fmt.Errorf("connection refused"))

	_, err := repo.Save(ctx, "abc12345", "https://ya.ru", "user1")

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- GetByShortURL ---

func TestSQLURLRepository_GetByShortURL_Found(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMockRepo(t)

	rows := pgxmock.NewRows([]string{"id", "short_url", "original_url", "user_id", "is_deleted"}).
		AddRow("550e8400-e29b-41d4-a716-446655440000", "abc12345", "https://ya.ru", "user1", false)

	mock.ExpectQuery(`SELECT id, short_url, original_url, user_id, is_deleted FROM urls WHERE short_url`).
		WithArgs("abc12345").
		WillReturnRows(rows)

	rec, err := repo.GetByShortURL(ctx, "abc12345")

	require.NoError(t, err)
	assert.Equal(t, "https://ya.ru", rec.OriginalURL)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLURLRepository_GetByShortURL_NotFound(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(`SELECT id, short_url, original_url, user_id, is_deleted FROM urls WHERE short_url`).
		WithArgs("notexists").
		WillReturnRows(pgxmock.NewRows([]string{"id", "short_url", "original_url", "user_id", "is_deleted"}))

	_, err := repo.GetByShortURL(ctx, "notexists")

	assert.ErrorIs(t, err, repository.ErrURLNotFound)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLURLRepository_GetByShortURL_Deleted(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMockRepo(t)

	rows := pgxmock.NewRows([]string{"id", "short_url", "original_url", "user_id", "is_deleted"}).
		AddRow("550e8400-e29b-41d4-a716-446655440000", "abc12345", "https://ya.ru", "user1", true)

	mock.ExpectQuery(`SELECT id, short_url, original_url, user_id, is_deleted FROM urls WHERE short_url`).
		WithArgs("abc12345").
		WillReturnRows(rows)

	_, err := repo.GetByShortURL(ctx, "abc12345")

	assert.ErrorIs(t, err, repository.ErrURLDeleted)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLURLRepository_GetByShortURL_Error(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(`SELECT id, short_url, original_url, user_id, is_deleted FROM urls WHERE short_url`).
		WithArgs("abc12345").
		WillReturnError(fmt.Errorf("db error"))

	_, err := repo.GetByShortURL(ctx, "abc12345")

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- GetByUserID ---

func TestSQLURLRepository_GetByUserID(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMockRepo(t)

	rows := pgxmock.NewRows([]string{"id", "short_url", "original_url", "user_id", "is_deleted"}).
		AddRow("550e8400-e29b-41d4-a716-446655440000", "s1", "https://ya.ru", "user1", false).
		AddRow("550e8400-e29b-41d4-a716-446655440001", "s2", "https://google.com", "user1", false)

	mock.ExpectQuery(`SELECT id, short_url, original_url, user_id, is_deleted FROM urls WHERE user_id`).
		WithArgs("user1").
		WillReturnRows(rows)

	records, err := repo.GetByUserID(ctx, "user1")

	require.NoError(t, err)
	assert.Len(t, records, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLURLRepository_GetByUserID_Error(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(`SELECT id, short_url, original_url, user_id, is_deleted FROM urls WHERE user_id`).
		WithArgs("user1").
		WillReturnError(fmt.Errorf("db error"))

	_, err := repo.GetByUserID(ctx, "user1")

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- DeleteURLs ---

func TestSQLURLRepository_DeleteURLs(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMockRepo(t)

	mock.ExpectExec(`UPDATE urls`).
		WithArgs(pgxmock.AnyArg(), "user1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 2))

	err := repo.DeleteURLs(ctx, []string{"s1", "s2"}, "user1")

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLURLRepository_DeleteURLs_Empty(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMockRepo(t)

	err := repo.DeleteURLs(ctx, []string{}, "user1")

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLURLRepository_DeleteURLs_Error(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMockRepo(t)

	mock.ExpectExec(`UPDATE urls`).
		WithArgs(pgxmock.AnyArg(), "user1").
		WillReturnError(fmt.Errorf("db error"))

	err := repo.DeleteURLs(ctx, []string{"s1"}, "user1")

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- SaveBatch ---

func TestSQLURLRepository_SaveBatch(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMockRepo(t)

	mock.ExpectBegin()
	rows1 := pgxmock.NewRows([]string{"id", "short_url", "original_url", "user_id", "is_deleted"}).
		AddRow("550e8400-e29b-41d4-a716-446655440000", "s1", "https://ya.ru", "user1", false)
	mock.ExpectQuery(`INSERT INTO urls`).
		WithArgs("s1", "https://ya.ru", "user1").
		WillReturnRows(rows1)
	rows2 := pgxmock.NewRows([]string{"id", "short_url", "original_url", "user_id", "is_deleted"}).
		AddRow("550e8400-e29b-41d4-a716-446655440001", "s2", "https://google.com", "user1", false)
	mock.ExpectQuery(`INSERT INTO urls`).
		WithArgs("s2", "https://google.com", "user1").
		WillReturnRows(rows2)
	mock.ExpectCommit()

	records := []model.URLRecord{
		{ShortURL: "s1", OriginalURL: "https://ya.ru", UserID: "user1"},
		{ShortURL: "s2", OriginalURL: "https://google.com", UserID: "user1"},
	}
	saved, err := repo.SaveBatch(ctx, records)

	require.NoError(t, err)
	assert.Len(t, saved, 2)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLURLRepository_SaveBatch_BeginError(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMockRepo(t)

	mock.ExpectBegin().WillReturnError(fmt.Errorf("begin error"))

	_, err := repo.SaveBatch(ctx, []model.URLRecord{{ShortURL: "s1", OriginalURL: "https://ya.ru", UserID: "user1"}})

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLURLRepository_SaveBatch_InsertError(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMockRepo(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO urls`).
		WithArgs("s1", "https://ya.ru", "user1").
		WillReturnError(fmt.Errorf("insert error"))
	mock.ExpectRollback()

	_, err := repo.SaveBatch(ctx, []model.URLRecord{{ShortURL: "s1", OriginalURL: "https://ya.ru", UserID: "user1"}})

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- Stats ---

func TestSQLURLRepository_Stats(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMockRepo(t)

	rows := pgxmock.NewRows([]string{"url_count", "user_count"}).AddRow(5, 3)
	mock.ExpectQuery(`SELECT`).WillReturnRows(rows)

	urlCount, userCount, err := repo.Stats(ctx)

	require.NoError(t, err)
	assert.Equal(t, 5, urlCount)
	assert.Equal(t, 3, userCount)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSQLURLRepository_Stats_Error(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMockRepo(t)

	mock.ExpectQuery(`SELECT`).WillReturnError(fmt.Errorf("db error"))

	_, _, err := repo.Stats(ctx)

	assert.Error(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// --- Ping ---

func TestSQLURLRepository_Ping(t *testing.T) {
	ctx := context.Background()
	repo, mock := newMockRepo(t)

	mock.ExpectPing()

	err := repo.Ping(ctx)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
