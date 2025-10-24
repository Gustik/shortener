package handler

import (
	"net/http"
)

func SetupRoutes(handler *URLHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" && r.Method == http.MethodPost {
			handler.ShortenURL(w, r)
			return
		}

		if r.Method == http.MethodGet {
			handler.GetOriginalURL(w, r)
			return
		}

		http.Error(w, "Bad request", http.StatusBadRequest)
	})

	return mux
}
