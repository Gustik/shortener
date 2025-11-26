package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Gustik/shortener/internal/zaplog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Тестовый handler, возвращающий простой текст
func testHandler(statusCode int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(statusCode)
		if body != "" {
			w.Write([]byte(body))
		}
	}
}

// Вспомогательная функция для распаковки gzip
func gunzip(data []byte) (string, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer reader.Close()

	result, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

// Клиент поддерживает gzip + ответ 2xx с телом, сжимаем
func TestGzipMiddleware_CompressResponse(t *testing.T) {
	middleware := GzipMiddleware(zaplog.NewNoop())
	handler := middleware(testHandler(http.StatusOK, "Hello, World!"))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, "Проверяем статус")
	assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"), "Проверяем заголовок Content-Encoding")

	decompressed, err := gunzip(rec.Body.Bytes())
	require.NoError(t, err, "failed to decompress")

	assert.Equal(t, "Hello, World!", decompressed)
}

// Клиент не поддерживает gzip, не сжимаем
func TestGzipMiddleware_NoCompressionWithoutAcceptEncoding(t *testing.T) {
	middleware := GzipMiddleware(zaplog.NewNoop())
	handler := middleware(testHandler(http.StatusOK, "Hello, World!"))

	req := httptest.NewRequest("GET", "/", nil)
	// НЕ устанавливаем Accept-Encoding

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Empty(t, rec.Header().Get("Content-Encoding"), "Content-Encoding должен отсутствовать")
	assert.Equal(t, "Hello, World!", rec.Body.String(), "Тело должно быть несжатым")
}

// Редирект 3xx не сжимаем, нет тела
func TestGzipMiddleware_NoCompressionForRedirect(t *testing.T) {
	redirectHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "https://example.com")
		w.WriteHeader(http.StatusTemporaryRedirect)
		// не пишем тело
	})

	middleware := GzipMiddleware(zaplog.NewNoop())
	handler := middleware(redirectHandler)

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTemporaryRedirect, rec.Code, "Проверяем статус редиректа")
	assert.Empty(t, rec.Header().Get("Content-Encoding"), "Content-Encoding не должен быть установлен для редиректа")
	assert.Zero(t, rec.Body.Len(), "Тело должно быть пустым для редиректа")
}

// Ошибка 4xx с телом, сжимаем
func TestGzipMiddleware_CompressionFor4xx(t *testing.T) {
	middleware := GzipMiddleware(zaplog.NewNoop())
	handler := middleware(testHandler(http.StatusNotFound, "Not Found"))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "gzip", rec.Header().Get("Content-Encoding"), "Content-Encoding должен быть установлен для 4xx с телом")

	decompressed, err := gunzip(rec.Body.Bytes())
	require.NoError(t, err, "failed to decompress")
	assert.Equal(t, "Not Found", decompressed, "Тело должно быть сжатым")
}

// Клиент отправляет сжатый запрос
func TestGzipMiddleware_DecompressRequest(t *testing.T) {
	// Handler читает тело запроса
	echoHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err, "Ошибка чтения тела запроса")
		w.Write(body)
	})

	middleware := GzipMiddleware(zaplog.NewNoop())
	handler := middleware(echoHandler)

	// Создаём сжатое тело
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, err := gw.Write([]byte("compressed data"))
	require.NoError(t, err, "Ошибка записи в gzip writer")
	require.NoError(t, gw.Close(), "Ошибка закрытия gzip writer")

	req := httptest.NewRequest("POST", "/", &buf)
	req.Header.Set("Content-Encoding", "gzip")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, "compressed data", rec.Body.String(), "Handler должен получить распакованные данные")
}
