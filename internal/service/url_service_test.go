package service_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"

	"github.com/Gustik/shortener/internal/repository"
	"github.com/Gustik/shortener/internal/service"
	"github.com/Gustik/shortener/internal/zaplog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestURLService_GetOriginalURL_Deleted(t *testing.T) {
	ctx := context.Background()
	repo := repository.NewInMemoryURLRepository()
	svc := service.NewURLService(repo, "http://localhost", zaplog.NewNoop())

	userID := "user123"
	shortID := "short1"

	// Создаём URL
	_, err := repo.Save(ctx, shortID, "https://example.com", userID)
	require.NoError(t, err)

	// Проверяем, что URL доступен
	url, err := svc.GetOriginalURL(ctx, shortID)
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com", url)

	// Удаляем URL
	err = repo.DeleteURLs(ctx, []string{shortID}, userID)
	require.NoError(t, err)

	// Проверяем, что URL возвращает ошибку ErrURLDeleted
	_, err = svc.GetOriginalURL(ctx, shortID)
	assert.ErrorIs(t, err, service.ErrURLDeleted)
}

func TestURLService_DeleteURLs(t *testing.T) {
	t.Run("Асинхронное удаление URL", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx := context.Background()
			repo := repository.NewInMemoryURLRepository()
			svc := service.NewURLService(repo, "http://localhost", zaplog.NewNoop())

			userID := "user123"

			// Создаём URL
			urls := []string{"short1", "short2", "short3"}
			for _, shortID := range urls {
				_, err := repo.Save(ctx, shortID, "https://example.com/"+shortID, userID)
				require.NoError(t, err)
			}

			// Удаляем URL асинхронно
			svc.DeleteURLs(ctx, userID, urls)

			// synctest.Wait() автоматически ждёт завершения всех горутин
			synctest.Wait()

			// Проверяем, что URL удалены
			for _, shortID := range urls {
				_, err := repo.GetByShortURL(ctx, shortID)
				assert.ErrorIs(t, err, repository.ErrURLDeleted, "URL %s должен быть удалён", shortID)
			}
		})
	})

	t.Run("Пустой список URL", func(t *testing.T) {
		ctx := context.Background()
		repo := repository.NewInMemoryURLRepository()
		svc := service.NewURLService(repo, "http://localhost", zaplog.NewNoop())

		// Не должно быть паники
		svc.DeleteURLs(ctx, "user123", []string{})
	})

	t.Run("Batch удаление большого количества URL", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			ctx := context.Background()
			repo := repository.NewInMemoryURLRepository()
			svc := service.NewURLService(repo, "http://localhost", zaplog.NewNoop())

			userID := "user123"
			count := 50

			// Создаём много URL
			urls := make([]string, count)
			for i := 0; i < count; i++ {
				shortID := "short" + string(rune('A'+i%26)) + string(rune('0'+i/26))
				urls[i] = shortID
				_, err := repo.Save(ctx, shortID, "https://example.com/"+shortID, userID)
				require.NoError(t, err)
			}

			// Удаляем URL асинхронно
			svc.DeleteURLs(ctx, userID, urls)

			// synctest.Wait() ждёт завершения всех горутин
			synctest.Wait()

			// Проверяем, что все URL удалены
			deletedCount := 0
			for _, shortID := range urls {
				_, err := repo.GetByShortURL(ctx, shortID)
				if errors.Is(err, repository.ErrURLDeleted) {
					deletedCount++
				}
			}

			assert.Equal(t, count, deletedCount, "Все URL должны быть удалены")
		})
	})
}
