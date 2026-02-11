package repository_test

import (
	"context"
	"testing"

	"github.com/Gustik/shortener/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryURLRepository_DeleteURLs(t *testing.T) {
	ctx := context.Background()

	t.Run("Успешное удаление URL", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()
		userID := "user123"

		// Создаём URL
		_, err := repo.Save(ctx, "short1", "https://example1.com", userID)
		require.NoError(t, err)
		_, err = repo.Save(ctx, "short2", "https://example2.com", userID)
		require.NoError(t, err)

		// Удаляем URL
		err = repo.DeleteURLs(ctx, []string{"short1"}, userID)
		assert.NoError(t, err)

		// Проверяем, что URL удалён
		_, err = repo.GetByShortURL(ctx, "short1")
		assert.ErrorIs(t, err, repository.ErrURLDeleted)

		// Проверяем, что второй URL всё ещё доступен
		record, err := repo.GetByShortURL(ctx, "short2")
		assert.NoError(t, err)
		assert.Equal(t, "https://example2.com", record.OriginalURL)
	})

	t.Run("Удаление URL другого пользователя не срабатывает", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()
		user1 := "user1"
		user2 := "user2"

		// Создаём URL для user1
		_, err := repo.Save(ctx, "short1", "https://example.com", user1)
		require.NoError(t, err)

		// Пытаемся удалить URL от имени user2
		err = repo.DeleteURLs(ctx, []string{"short1"}, user2)
		assert.NoError(t, err)

		// Проверяем, что URL всё ещё доступен
		record, err := repo.GetByShortURL(ctx, "short1")
		assert.NoError(t, err)
		assert.Equal(t, "https://example.com", record.OriginalURL)
		assert.False(t, record.IsDeleted)
	})

	t.Run("Удаление нескольких URL", func(t *testing.T) {
		repo := repository.NewInMemoryURLRepository()
		userID := "user123"

		// Создаём несколько URL
		for i := 1; i <= 5; i++ {
			_, err := repo.Save(ctx, "short"+string(rune('0'+i)), "https://example"+string(rune('0'+i))+".com", userID)
			require.NoError(t, err)
		}

		// Удаляем первые 3 URL
		err := repo.DeleteURLs(ctx, []string{"short1", "short2", "short3"}, userID)
		assert.NoError(t, err)

		// Проверяем, что удалённые URL недоступны
		for i := 1; i <= 3; i++ {
			_, err := repo.GetByShortURL(ctx, "short"+string(rune('0'+i)))
			assert.ErrorIs(t, err, repository.ErrURLDeleted)
		}

		// Проверяем, что оставшиеся URL доступны
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
