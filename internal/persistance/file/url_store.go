package file

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/nestjam/yap-shortener/internal/domain"
)

type URLStore struct {
	encoder *json.Encoder
	urls    []StoredURL
}

type StoredURL struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	ID          int    `json:"uuid"`
}

func New(rw io.ReadWriter) (*URLStore, error) {
	urls, err := getURLs(rw)

	if err != nil {
		return nil, fmt.Errorf("new file storage: %w", err)
	}

	return &URLStore{
		json.NewEncoder(rw),
		urls,
	}, nil
}

func getURLs(rw io.ReadWriter) ([]StoredURL, error) {
	dec := json.NewDecoder(rw)
	var urls []StoredURL

	for dec.More() {
		var url StoredURL
		err := dec.Decode(&url)

		if err != nil {
			return nil, fmt.Errorf("get URLs: %w", err)
		}

		urls = append(urls, url)
	}

	return urls, nil
}

func (u *URLStore) Get(shortURL string) (string, error) {
	if url, ok := u.find(shortURL); ok {
		return url.OriginalURL, nil
	}

	return "", domain.ErrURLNotFound
}

func (u *URLStore) find(shortURL string) (*StoredURL, bool) {
	for i := 0; i < len(u.urls); i++ {
		if u.urls[i].ShortURL == shortURL {
			return &u.urls[i], true
		}
	}
	return nil, false
}

func (u *URLStore) Add(shortURL, originalURL string) error {
	url := StoredURL{
		ID:          len(u.urls),
		ShortURL:    shortURL,
		OriginalURL: originalURL,
	}
	u.urls = append(u.urls, url)
	err := u.encoder.Encode(url)

	if err != nil {
		return fmt.Errorf("add url: %w", err)
	}

	return nil
}

func (u *URLStore) IsAvailable() bool {
	return true
}
