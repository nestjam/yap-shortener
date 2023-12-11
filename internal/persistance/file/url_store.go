package file

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/nestjam/yap-shortener/internal/domain"
	"github.com/nestjam/yap-shortener/internal/persistance/inmemory"
	"github.com/pkg/errors"
)

type URLStore struct {
	encoder *json.Encoder
	s       *inmemory.URLStore
	id      int
	mu      sync.Mutex
}

type StoredURL struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	ID          int    `json:"uuid"`
}

func New(rw io.ReadWriter) (*URLStore, error) {
	const op = "new file storage"
	urls, err := readURLs(rw)

	if err != nil {
		return nil, errors.Wrap(err, op)
	}

	s := &inmemory.URLStore{}
	ctx := context.Background()
	for i := 0; i < len(urls); i++ {
		err := s.AddURL(ctx, urls[i].ShortURL, urls[i].OriginalURL)

		if err != nil {
			return nil, errors.Wrap(err, op)
		}
	}

	return &URLStore{
		encoder: json.NewEncoder(rw),
		s:       &inmemory.URLStore{},
	}, nil
}

func readURLs(rw io.ReadWriter) ([]StoredURL, error) {
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

func (u *URLStore) GetOriginalURL(ctx context.Context, shortURL string) (string, error) {
	const op = "add"

	originalURL, err := u.s.GetOriginalURL(ctx, shortURL)

	if err != nil {
		return "", errors.Wrap(err, op)
	}

	return originalURL, nil
}

func (u *URLStore) AddURL(ctx context.Context, shortURL, originalURL string) error {
	const op = "add"

	err := u.s.AddURL(ctx, shortURL, originalURL)

	if err != nil {
		return errors.Wrap(err, op)
	}

	u.mu.Lock()
	defer u.mu.Unlock()

	url := StoredURL{
		ID:          u.id,
		ShortURL:    shortURL,
		OriginalURL: originalURL,
	}
	err = u.encoder.Encode(url)
	u.id++

	if err != nil {
		return errors.Wrap(err, op)
	}

	return nil
}

func (u *URLStore) AddURLs(ctx context.Context, pairs []domain.URLPair) error {
	const op = "add batch"

	err := u.s.AddURLs(ctx, pairs)

	if err != nil {
		return errors.Wrap(err, op)
	}

	u.mu.Lock()
	defer u.mu.Unlock()

	for i := 0; i < len(pairs); i++ {
		url := StoredURL{
			ID:          u.id,
			ShortURL:    pairs[i].ShortURL,
			OriginalURL: pairs[i].OriginalURL,
		}
		err = u.encoder.Encode(url)
		u.id++

		if err != nil {
			return errors.Wrap(err, op)
		}
	}

	return nil
}

func (u *URLStore) IsAvailable(ctx context.Context) bool {
	return true
}
