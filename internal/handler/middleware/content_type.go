package middleware

import "net/http"

// ContentTypeMiddleware возвращает middleware, который отклоняет запросы,
// чей заголовок Content-Type не совпадает с ожидаемым значением.
func ContentTypeMiddleware(contentType string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Content-Type") != contentType {
				http.Error(w, "Invalid content type", http.StatusBadRequest)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
