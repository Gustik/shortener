package model

import "github.com/google/uuid"

// Request представляет JSON-тело запроса для эндпоинта сокращения URL.
type Request struct {
	URL string `json:"url"`
}

// Response представляет JSON-тело ответа с сокращённым URL.
type Response struct {
	Result string `json:"result"`
}

// BatchRequest представляет один элемент в пакетном запросе на сокращение URL.
type BatchRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// BatchResponse представляет один элемент в ответе на пакетное сокращение URL.
type BatchResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

// UserURLResponse представляет сокращённый URL, принадлежащий пользователю.
type UserURLResponse struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

// URLRecord — запись, связывающая короткий URL с оригинальным.
type URLRecord struct {
	UUID        uuid.UUID `json:"uuid"`
	ShortURL    string    `json:"short_url"`
	OriginalURL string    `json:"original_url"`
	UserID      string    `json:"user_id"`
	IsDeleted   bool      `json:"is_deleted"`
}

// NextID присваивает записи новый случайный UUID.
func (u *URLRecord) NextID() {
	u.UUID = uuid.New()
}

// AuditEvent представляет одну запись журнала аудита операций с URL.
type AuditEvent struct {
	Timestamp int64  `json:"ts"`
	Action    string `json:"action"`
	UserID    string `json:"user_id"`
	URL       string `json:"url"`
}
