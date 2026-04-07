package repository_test

import (
	"context"
	"testing"

	"github.com/Gustik/shortener/internal/model"
	"github.com/Gustik/shortener/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Save ---

func TestInMemoryURLRepository_Save(t *testing.T) {
	ctx := context.Background()

	t.Run("Сохраняет новую запись", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()

		rec, err := repo.Save(ctx, "abc12345", "https://ya.ru", "user1")

		require.NoError(t, err)
		assert.Equal(t, "abc12345", rec.ShortURL)
		assert.Equal(t, "https://ya.ru", rec.OriginalURL)
		assert.Equal(t, "user1", rec.UserID)
		assert.NotEmpty(t, rec.UUID)
	})

	t.Run("Конфликт по оригинальному URL возвращает ErrURLConflict", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()
		_, err := repo.Save(ctx, "abc12345", "https://ya.ru", "user1")
		require.NoError(t, err)

		rec, err := repo.Save(ctx, "xyz99999", "https://ya.ru", "user1")

		assert.ErrorIs(t, err, repository.ErrURLConflict)
		assert.Equal(t, "abc12345", rec.ShortURL) // возвращает существующую запись
	})

	t.Run("Конфликт по короткому URL возвращает ErrShortURLConflict", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()
		_, err := repo.Save(ctx, "abc12345", "https://ya.ru", "user1")
		require.NoError(t, err)

		rec, err := repo.Save(ctx, "abc12345", "https://google.com", "user1")

		assert.ErrorIs(t, err, repository.ErrShortURLConflict)
		assert.Nil(t, rec)
	})
}

// --- GetByShortURL ---

func TestInMemoryURLRepository_GetByShortURL(t *testing.T) {
	ctx := context.Background()

	t.Run("Возвращает существующую запись", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()
		_, err := repo.Save(ctx, "abc12345", "https://ya.ru", "user1")
		require.NoError(t, err)

		rec, err := repo.GetByShortURL(ctx, "abc12345")

		require.NoError(t, err)
		assert.Equal(t, "https://ya.ru", rec.OriginalURL)
	})

	t.Run("Несуществующий shortURL возвращает ErrURLNotFound", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()

		_, err := repo.GetByShortURL(ctx, "notexists")

		assert.ErrorIs(t, err, repository.ErrURLNotFound)
	})

	t.Run("Удалённый URL возвращает ErrURLDeleted", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()
		_, err := repo.Save(ctx, "abc12345", "https://ya.ru", "user1")
		require.NoError(t, err)
		err = repo.DeleteURLs(ctx, []string{"abc12345"}, "user1")
		require.NoError(t, err)

		_, err = repo.GetByShortURL(ctx, "abc12345")

		assert.ErrorIs(t, err, repository.ErrURLDeleted)
	})
}

// --- SaveBatch ---

func TestInMemoryURLRepository_SaveBatch(t *testing.T) {
	ctx := context.Background()

	t.Run("Сохраняет несколько записей", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()
		records := []model.URLRecord{
			{ShortURL: "aaa11111", OriginalURL: "https://ya.ru", UserID: "user1"},
			{ShortURL: "bbb22222", OriginalURL: "https://google.com", UserID: "user1"},
		}

		saved, err := repo.SaveBatch(ctx, records)

		require.NoError(t, err)
		require.Len(t, saved, 2)
		assert.Equal(t, "aaa11111", saved[0].ShortURL)
		assert.Equal(t, "bbb22222", saved[1].ShortURL)
	})

	t.Run("Дубль оригинального URL возвращает существующую запись без ошибки", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()
		_, err := repo.Save(ctx, "existing1", "https://ya.ru", "user1")
		require.NoError(t, err)

		saved, err := repo.SaveBatch(ctx, []model.URLRecord{
			{ShortURL: "newshort1", OriginalURL: "https://ya.ru", UserID: "user1"},
		})

		require.NoError(t, err)
		require.Len(t, saved, 1)
		assert.Equal(t, "existing1", saved[0].ShortURL) // вернул существующую
	})

	t.Run("Конфликт по короткому URL возвращает ErrShortURLConflict", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()
		_, err := repo.Save(ctx, "aaa11111", "https://ya.ru", "user1")
		require.NoError(t, err)

		_, err = repo.SaveBatch(ctx, []model.URLRecord{
			{ShortURL: "aaa11111", OriginalURL: "https://google.com", UserID: "user1"},
		})

		assert.ErrorIs(t, err, repository.ErrShortURLConflict)
	})
}

// --- GetByUserID ---

func TestInMemoryURLRepository_GetByUserID(t *testing.T) {
	ctx := context.Background()

	t.Run("Возвращает только URL указанного пользователя", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()
		_, err := repo.Save(ctx, "u1short1", "https://ya.ru", "user1")
		require.NoError(t, err)
		_, err = repo.Save(ctx, "u1short2", "https://google.com", "user1")
		require.NoError(t, err)
		_, err = repo.Save(ctx, "u2short1", "https://other.com", "user2")
		require.NoError(t, err)

		records, err := repo.GetByUserID(ctx, "user1")

		require.NoError(t, err)
		require.Len(t, records, 2)
		for _, r := range records {
			assert.Equal(t, "user1", r.UserID)
		}
	})

	t.Run("Пользователь без URL возвращает пустой срез", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()

		records, err := repo.GetByUserID(ctx, "nobody")

		require.NoError(t, err)
		assert.Empty(t, records)
	})
}

// --- DeleteURLs ---

func TestInMemoryURLRepository_DeleteURLs(t *testing.T) {
	ctx := context.Background()

	t.Run("Успешное удаление URL", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()
		userID := "user123"

		_, err := repo.Save(ctx, "short1", "https://example1.com", userID)
		require.NoError(t, err)
		_, err = repo.Save(ctx, "short2", "https://example2.com", userID)
		require.NoError(t, err)

		err = repo.DeleteURLs(ctx, []string{"short1"}, userID)
		assert.NoError(t, err)

		_, err = repo.GetByShortURL(ctx, "short1")
		assert.ErrorIs(t, err, repository.ErrURLDeleted)

		record, err := repo.GetByShortURL(ctx, "short2")
		assert.NoError(t, err)
		assert.Equal(t, "https://example2.com", record.OriginalURL)
	})

	t.Run("Удаление URL другого пользователя не срабатывает", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()

		_, err := repo.Save(ctx, "short1", "https://example.com", "user1")
		require.NoError(t, err)

		err = repo.DeleteURLs(ctx, []string{"short1"}, "user2")
		assert.NoError(t, err)

		record, err := repo.GetByShortURL(ctx, "short1")
		assert.NoError(t, err)
		assert.False(t, record.IsDeleted)
	})

	t.Run("Удаление нескольких URL", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()
		userID := "user123"

		for i := 1; i <= 5; i++ {
			_, err := repo.Save(ctx, "short"+string(rune('0'+i)), "https://example"+string(rune('0'+i))+".com", userID)
			require.NoError(t, err)
		}

		err := repo.DeleteURLs(ctx, []string{"short1", "short2", "short3"}, userID)
		assert.NoError(t, err)

		for i := 1; i <= 3; i++ {
			_, err := repo.GetByShortURL(ctx, "short"+string(rune('0'+i)))
			assert.ErrorIs(t, err, repository.ErrURLDeleted)
		}
		for i := 4; i <= 5; i++ {
			record, err := repo.GetByShortURL(ctx, "short"+string(rune('0'+i)))
			assert.NoError(t, err)
			assert.False(t, record.IsDeleted)
		}
	})

	t.Run("Удаление пустого списка URL", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()
		err := repo.DeleteURLs(ctx, []string{}, "user123")
		assert.NoError(t, err)
	})
}

// --- Stats ---

func TestInMemoryURLRepository_Stats(t *testing.T) {
	ctx := context.Background()

	t.Run("Считает только неудалённые URL и уникальных пользователей", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()
		_, err := repo.Save(ctx, "u1s1", "https://ya.ru", "user1")
		require.NoError(t, err)
		_, err = repo.Save(ctx, "u1s2", "https://google.com", "user1")
		require.NoError(t, err)
		_, err = repo.Save(ctx, "u2s1", "https://other.com", "user2")
		require.NoError(t, err)

		// Удаляем один URL
		err = repo.DeleteURLs(ctx, []string{"u1s2"}, "user1")
		require.NoError(t, err)

		urlCount, userCount, err := repo.Stats(ctx)

		require.NoError(t, err)
		assert.Equal(t, 2, urlCount)  // 3 сохранено, 1 удалено
		assert.Equal(t, 2, userCount) // user1 и user2
	})

	t.Run("Пустой репозиторий", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()

		urlCount, userCount, err := repo.Stats(ctx)

		require.NoError(t, err)
		assert.Equal(t, 0, urlCount)
		assert.Equal(t, 0, userCount)
	})
}

// --- Ping ---

func TestInMemoryURLRepository_Ping(t *testing.T) {
	repo := repository.NewInMemoryURLRepository()
	err := repo.Ping(context.Background())
	assert.NoError(t, err)
}
