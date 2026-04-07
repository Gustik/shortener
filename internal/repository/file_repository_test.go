package repository_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/Gustik/shortener/internal/model"
	"github.com/Gustik/shortener/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func tempFile(t *testing.T) *os.File {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "repo-*.jsonl")
	require.NoError(t, err)
	return f
}

func TestFileURLRepository_Save(t *testing.T) {
	ctx := context.Background()

	t.Run("Сохраняет запись и персистирует её в файл", func(t *testing.T) {
		f := tempFile(t)
		repo, err := repository.NewFileURLRepository(f)
		require.NoError(t, err)

		rec, err := repo.Save(ctx, "abc12345", "https://ya.ru", "user1")

		require.NoError(t, err)
		assert.Equal(t, "abc12345", rec.ShortURL)

		// Проверяем, что запись попала в файл
		info, err := f.Stat()
		require.NoError(t, err)
		assert.Positive(t, info.Size())
	})

	t.Run("Конфликт по оригинальному URL возвращает существующую запись", func(t *testing.T) {
		f := tempFile(t)
		repo, err := repository.NewFileURLRepository(f)
		require.NoError(t, err)

		_, err = repo.Save(ctx, "abc12345", "https://ya.ru", "user1")
		require.NoError(t, err)

		rec, err := repo.Save(ctx, "xyz99999", "https://ya.ru", "user1")

		assert.ErrorIs(t, err, repository.ErrURLConflict)
		assert.Equal(t, "abc12345", rec.ShortURL)
	})

	t.Run("Конфликт по короткому URL возвращает ErrShortURLConflict", func(t *testing.T) {
		f := tempFile(t)
		repo, err := repository.NewFileURLRepository(f)
		require.NoError(t, err)

		_, err = repo.Save(ctx, "abc12345", "https://ya.ru", "user1")
		require.NoError(t, err)

		_, err = repo.Save(ctx, "abc12345", "https://google.com", "user1")

		assert.ErrorIs(t, err, repository.ErrShortURLConflict)
	})
}

func TestFileURLRepository_SaveBatch(t *testing.T) {
	ctx := context.Background()

	t.Run("Сохраняет несколько записей", func(t *testing.T) {
		f := tempFile(t)
		repo, err := repository.NewFileURLRepository(f)
		require.NoError(t, err)

		records := []model.URLRecord{
			{ShortURL: "aaa11111", OriginalURL: "https://ya.ru", UserID: "user1"},
			{ShortURL: "bbb22222", OriginalURL: "https://google.com", UserID: "user1"},
		}
		saved, err := repo.SaveBatch(ctx, records)

		require.NoError(t, err)
		require.Len(t, saved, 2)

		// Обе записи должны быть в файле
		info, err := f.Stat()
		require.NoError(t, err)
		assert.Positive(t, info.Size())
	})

	t.Run("Дубль оригинального URL не пишется в файл повторно", func(t *testing.T) {
		f := tempFile(t)
		repo, err := repository.NewFileURLRepository(f)
		require.NoError(t, err)

		_, err = repo.Save(ctx, "existing1", "https://ya.ru", "user1")
		require.NoError(t, err)

		saved, err := repo.SaveBatch(ctx, []model.URLRecord{
			{ShortURL: "newshort1", OriginalURL: "https://ya.ru", UserID: "user1"},
		})

		require.NoError(t, err)
		assert.Equal(t, "existing1", saved[0].ShortURL)
	})
}

func TestFileURLRepository_Persistence(t *testing.T) {
	ctx := context.Background()

	t.Run("Данные читаются из файла при повторном открытии", func(t *testing.T) {
		f := tempFile(t)
		path := f.Name()

		// Пишем данные
		repo1, err := repository.NewFileURLRepository(f)
		require.NoError(t, err)
		_, err = repo1.Save(ctx, "abc12345", "https://ya.ru", "user1")
		require.NoError(t, err)
		f.Close()

		// Открываем файл заново
		f2, err := os.OpenFile(path, os.O_RDWR, 0644)
		require.NoError(t, err)
		defer f2.Close()

		repo2, err := repository.NewFileURLRepository(f2)
		require.NoError(t, err)

		rec, err := repo2.GetByShortURL(ctx, "abc12345")
		require.NoError(t, err)
		assert.Equal(t, "https://ya.ru", rec.OriginalURL)
	})

	t.Run("Невалидный JSON в файле возвращает ошибку", func(t *testing.T) {
		f := tempFile(t)
		_, err := f.WriteString("not-json\n")
		require.NoError(t, err)
		_, err = f.Seek(0, 0)
		require.NoError(t, err)

		_, err = repository.NewFileURLRepository(f)
		assert.Error(t, err)
	})

	t.Run("Файл с существующими записями загружается корректно", func(t *testing.T) {
		f := tempFile(t)

		records := []model.URLRecord{
			{ShortURL: "s1", OriginalURL: "https://ya.ru", UserID: "u1"},
			{ShortURL: "s2", OriginalURL: "https://google.com", UserID: "u1"},
		}
		for _, r := range records {
			data, err := json.Marshal(r)
			require.NoError(t, err)
			_, err = f.Write(append(data, '\n'))
			require.NoError(t, err)
		}
		_, err := f.Seek(0, 0)
		require.NoError(t, err)

		repo, err := repository.NewFileURLRepository(f)
		require.NoError(t, err)

		rec, err := repo.GetByShortURL(ctx, "s1")
		require.NoError(t, err)
		assert.Equal(t, "https://ya.ru", rec.OriginalURL)
	})
}
