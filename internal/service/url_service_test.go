package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"testing/synctest"

	"github.com/Gustik/shortener/internal/model"
	"github.com/Gustik/shortener/internal/repository"
	"github.com/Gustik/shortener/internal/zaplog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// alwaysConflictRepo — стаб, всегда возвращающий ErrShortURLConflict из Save.
// Используется для тестирования ветки ErrMaxRetriesExceeded.
type alwaysConflictRepo struct{ repository.InMemoryURLRepository }

func (r *alwaysConflictRepo) Save(_ context.Context, shortURL, _, _ string) (*model.URLRecord, error) {
	return nil, fmt.Errorf("short URL '%s' already exists: %w", shortURL, repository.ErrShortURLConflict)
}

// --- ShortenURL ---

func TestURLService_ShortenURL(t *testing.T) {
	ctx := context.Background()

	t.Run("Успешное создание", func(t *testing.T) {
		svc := NewURLService(repository.NewInMemoryURLRepository(), "http://localhost", zaplog.NewNoop())

		result, err := svc.ShortenURL(ctx, "https://ya.ru", "user1")

		require.NoError(t, err)
		assert.Contains(t, result, "http://localhost/")
	})

	t.Run("Пустой URL возвращает ErrEmptyURL", func(t *testing.T) {
		svc := NewURLService(repository.NewInMemoryURLRepository(), "http://localhost", zaplog.NewNoop())

		_, err := svc.ShortenURL(ctx, "", "user1")

		assert.ErrorIs(t, err, ErrEmptyURL)
	})

	t.Run("Повторный URL возвращает ErrURLExists с тем же shortURL", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()
		svc := NewURLService(repo, "http://localhost", zaplog.NewNoop())

		first, err := svc.ShortenURL(ctx, "https://ya.ru", "user1")
		require.NoError(t, err)

		second, err := svc.ShortenURL(ctx, "https://ya.ru", "user1")

		assert.ErrorIs(t, err, ErrURLExists)
		assert.Equal(t, first, second)
	})

	t.Run("Исчерпание попыток возвращает ErrMaxRetriesExceeded", func(t *testing.T) {
		svc := NewURLService(&alwaysConflictRepo{}, "http://localhost", zaplog.NewNoop())

		_, err := svc.ShortenURL(ctx, "https://ya.ru", "user1")

		assert.ErrorIs(t, err, ErrMaxRetriesExceeded)
	})
}

// --- ShortenURLBatch ---

func TestURLService_ShortenURLBatch(t *testing.T) {
	ctx := context.Background()

	t.Run("Успешное создание batch", func(t *testing.T) {
		svc := NewURLService(repository.NewInMemoryURLRepository(), "http://localhost", zaplog.NewNoop())
		batch := []model.BatchRequest{
			{CorrelationID: "c1", OriginalURL: "https://ya.ru"},
			{CorrelationID: "c2", OriginalURL: "https://google.com"},
		}

		resp, err := svc.ShortenURLBatch(ctx, batch, "user1")

		require.NoError(t, err)
		require.Len(t, resp, 2)
		assert.Equal(t, "c1", resp[0].CorrelationID)
		assert.Equal(t, "c2", resp[1].CorrelationID)
		assert.Contains(t, resp[0].ShortURL, "http://localhost/")
		assert.Contains(t, resp[1].ShortURL, "http://localhost/")
	})

	t.Run("Пустой batch возвращает ErrEmptyURLBatch", func(t *testing.T) {
		svc := NewURLService(repository.NewInMemoryURLRepository(), "http://localhost", zaplog.NewNoop())

		_, err := svc.ShortenURLBatch(ctx, []model.BatchRequest{}, "user1")

		assert.ErrorIs(t, err, ErrEmptyURLBatch)
	})

	t.Run("Элемент с пустым OriginalURL возвращает ErrEmptyURL", func(t *testing.T) {
		svc := NewURLService(repository.NewInMemoryURLRepository(), "http://localhost", zaplog.NewNoop())

		_, err := svc.ShortenURLBatch(ctx, []model.BatchRequest{{CorrelationID: "c1", OriginalURL: ""}}, "user1")

		assert.ErrorIs(t, err, ErrEmptyURL)
	})
}

// --- GetOriginalURL ---

func TestURLService_GetOriginalURL(t *testing.T) {
	ctx := context.Background()

	t.Run("Возвращает оригинальный URL", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()
		svc := NewURLService(repo, "http://localhost", zaplog.NewNoop())
		_, err := repo.Save(ctx, "abc12345", "https://ya.ru", "user1")
		require.NoError(t, err)

		result, err := svc.GetOriginalURL(ctx, "abc12345")

		require.NoError(t, err)
		assert.Equal(t, "https://ya.ru", result)
	})

	t.Run("Пустой shortID возвращает ErrEmptyShortID", func(t *testing.T) {
		svc := NewURLService(repository.NewInMemoryURLRepository(), "http://localhost", zaplog.NewNoop())

		_, err := svc.GetOriginalURL(ctx, "")

		assert.ErrorIs(t, err, ErrEmptyShortID)
	})

	t.Run("Несуществующий shortID возвращает ErrURLNotFound", func(t *testing.T) {
		svc := NewURLService(repository.NewInMemoryURLRepository(), "http://localhost", zaplog.NewNoop())

		_, err := svc.GetOriginalURL(ctx, "notexists")

		assert.ErrorIs(t, err, ErrURLNotFound)
	})

	t.Run("Удалённый URL возвращает ErrURLDeleted", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()
		svc := NewURLService(repo, "http://localhost", zaplog.NewNoop())
		_, err := repo.Save(ctx, "abc12345", "https://ya.ru", "user1")
		require.NoError(t, err)
		err = repo.DeleteURLs(ctx, []string{"abc12345"}, "user1")
		require.NoError(t, err)

		_, err = svc.GetOriginalURL(ctx, "abc12345")

		assert.ErrorIs(t, err, ErrURLDeleted)
	})
}

// --- GetUserURLs ---

func TestURLService_GetUserURLs(t *testing.T) {
	ctx := context.Background()

	t.Run("Возвращает URL пользователя с полным shortURL", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()
		svc := NewURLService(repo, "http://localhost", zaplog.NewNoop())
		_, err := repo.Save(ctx, "abc12345", "https://ya.ru", "user1")
		require.NoError(t, err)
		_, err = repo.Save(ctx, "xyz99999", "https://google.com", "user2")
		require.NoError(t, err)

		urls, err := svc.GetUserURLs(ctx, "user1")

		require.NoError(t, err)
		require.Len(t, urls, 1)
		assert.Equal(t, "http://localhost/abc12345", urls[0].ShortURL)
		assert.Equal(t, "https://ya.ru", urls[0].OriginalURL)
	})

	t.Run("Пользователь без URL возвращает пустой срез", func(t *testing.T) {
		svc := NewURLService(repository.NewInMemoryURLRepository(), "http://localhost", zaplog.NewNoop())

		urls, err := svc.GetUserURLs(ctx, "nobody")

		require.NoError(t, err)
		assert.Empty(t, urls)
	})
}

// --- DeleteURLs ---

func TestURLService_DeleteURLs(t *testing.T) {
	t.Run("Асинхронное удаление URL", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx := context.Background()
			repo := repository.NewInMemoryURLRepository()
			svc := NewURLService(repo, "http://localhost", zaplog.NewNoop())

			userID := "user123"
			urls := []string{"short1", "short2", "short3"}
			for _, shortID := range urls {
				_, err := repo.Save(ctx, shortID, "https://example.com/"+shortID, userID)
				require.NoError(t, err)
			}

			svc.DeleteURLs(ctx, userID, urls)
			synctest.Wait()

			for _, shortID := range urls {
				_, err := repo.GetByShortURL(ctx, shortID)
				assert.ErrorIs(t, err, repository.ErrURLDeleted, "URL %s должен быть удалён", shortID)
			}
		})
	})

	t.Run("Пустой список URL не вызывает паники", func(t *testing.T) {
		ctx := context.Background()
		svc := NewURLService(repository.NewInMemoryURLRepository(), "http://localhost", zaplog.NewNoop())
		svc.DeleteURLs(ctx, "user123", []string{})
	})

	t.Run("Batch удаление большого количества URL", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx := context.Background()
			repo := repository.NewInMemoryURLRepository()
			svc := NewURLService(repo, "http://localhost", zaplog.NewNoop())

			userID := "user123"
			count := 50
			urls := make([]string, count)
			for i := 0; i < count; i++ {
				shortID := "short" + string(rune('A'+i%26)) + string(rune('0'+i/26))
				urls[i] = shortID
				_, err := repo.Save(ctx, shortID, "https://example.com/"+shortID, userID)
				require.NoError(t, err)
			}

			svc.DeleteURLs(ctx, userID, urls)
			synctest.Wait()

			deletedCount := 0
			for _, shortID := range urls {
				_, err := repo.GetByShortURL(ctx, shortID)
				if errors.Is(err, repository.ErrURLDeleted) {
					deletedCount++
				}
			}
			assert.Equal(t, count, deletedCount)
		})
	})
}

// --- Stats ---

func TestURLService_Stats(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewInMemoryURLRepository()
	svc := NewURLService(repo, "http://localhost", zaplog.NewNoop())

	_, err := repo.Save(ctx, "s1", "https://ya.ru", "user1")
	require.NoError(t, err)
	_, err = repo.Save(ctx, "s2", "https://google.com", "user2")
	require.NoError(t, err)

	urlCount, userCount, err := svc.Stats(ctx)

	require.NoError(t, err)
	assert.Equal(t, 2, urlCount)
	assert.Equal(t, 2, userCount)
}

// --- Ping ---

func TestURLService_Ping(t *testing.T) {
	svc := NewURLService(repository.NewInMemoryURLRepository(), "http://localhost", zaplog.NewNoop())
	err := svc.Ping(context.Background())
	assert.NoError(t, err)
}

// --- Benchmark ---

func BenchmarkGenerateShortURL(b *testing.B) {
	repo := repository.NewInMemoryURLRepository()
	svc := NewURLService(repo, "http://localhost", zaplog.NewNoop())
	for b.Loop() {
		svc.generateShortURL()
	}
}
