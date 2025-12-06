package model

import "github.com/google/uuid"

type Request struct {
	URL string `json:"url"`
}

type Response struct {
	Result string `json:"result"`
}

type URLRecord struct {
	UUID        uuid.UUID `json:"uuid"`
	ShortURL    string    `json:"short_url"`
	OriginalURL string    `json:"original_url"`
}

func (u *URLRecord) NextID() {
	u.UUID = uuid.New()
}
