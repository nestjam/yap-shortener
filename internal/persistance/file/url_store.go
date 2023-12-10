package file

import (
	"context"
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

func (u *URLStore) Get(ctx context.Context, shortURL string) (string, error) {
	if url, ok := u.findOriginalURL(shortURL); ok {
		return url, nil
	}

	return "", domain.ErrOriginalURLNotFound
}

func (u *URLStore) findOriginalURL(shortURL string) (string, bool) {
	for i := 0; i < len(u.urls); i++ {
		if u.urls[i].ShortURL == shortURL {
			return u.urls[i].OriginalURL, true
		}
	}
	return "", false
}

func (u *URLStore) Add(ctx context.Context, shortURL, originalURL string) error {
	if shortURL, ok := u.findShortURL(originalURL); ok {
		return domain.NewOriginalURLExistsError(shortURL, nil)
	}

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

func (u *URLStore) findShortURL(originalURL string) (string, bool) {
	for i := 0; i < len(u.urls); i++ {
		if u.urls[i].OriginalURL == originalURL {
			return u.urls[i].ShortURL, true
		}
	}
	return "", false
}

func (u *URLStore) AddBatch(ctx context.Context, pairs []domain.URLPair) error {
	for _, p := range pairs {
		err := u.Add(ctx, p.ShortURL, p.OriginalURL)

		if err != nil {
			return fmt.Errorf("add batch: %w", err)
		}
	}
	return nil
}

func (u *URLStore) IsAvailable(ctx context.Context) bool {
	return true
}
