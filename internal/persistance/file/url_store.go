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

type FileURLStore struct {
	encoder *json.Encoder
	s       *inmemory.InmemoryURLStore
	id      int
	mu      sync.Mutex
}

type StoredURL struct {
	ShortURL    string        `json:"short_url"`
	OriginalURL string        `json:"original_url"`
	ID          int           `json:"uuid"`
	UserID      domain.UserID `json:"user_id"`
}

func New(ctx context.Context, rw io.ReadWriter) (*FileURLStore, error) {
	const op = "new file storage"
	records, err := readURLs(rw)

	if err != nil {
		return nil, errors.Wrap(err, op)
	}

	s := &inmemory.InmemoryURLStore{}
	for i := 0; i < len(records); i++ {
		rec := records[i]
		pair := domain.URLPair{
			ShortURL:    rec.ShortURL,
			OriginalURL: rec.OriginalURL,
		}
		err := s.AddURL(ctx, pair, rec.UserID)

		if err != nil {
			return nil, errors.Wrap(err, op)
		}
	}

	return &FileURLStore{
		encoder: json.NewEncoder(rw),
		s:       &inmemory.InmemoryURLStore{},
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

func (u *FileURLStore) GetOriginalURL(ctx context.Context, shortURL string) (string, error) {
	const op = "get original URL"
	originalURL, err := u.s.GetOriginalURL(ctx, shortURL)

	if err != nil {
		return "", errors.Wrap(err, op)
	}

	return originalURL, nil
}

func (u *FileURLStore) AddURL(ctx context.Context, pair domain.URLPair, userID domain.UserID) error {
	const op = "add URL"
	err := u.s.AddURL(ctx, pair, userID)

	if err != nil {
		return errors.Wrap(err, op)
	}

	u.mu.Lock()
	defer u.mu.Unlock()

	url := StoredURL{
		ID:          u.id,
		ShortURL:    pair.ShortURL,
		OriginalURL: pair.OriginalURL,
		UserID:      userID,
	}
	err = u.encoder.Encode(url)
	u.id++

	if err != nil {
		return errors.Wrap(err, op)
	}

	return nil
}

func (u *FileURLStore) AddURLs(ctx context.Context, pairs []domain.URLPair, userID domain.UserID) error {
	_ = u.s.AddURLs(ctx, pairs, userID)

	u.mu.Lock()
	defer u.mu.Unlock()

	for i := 0; i < len(pairs); i++ {
		url := StoredURL{
			ID:          u.id,
			ShortURL:    pairs[i].ShortURL,
			OriginalURL: pairs[i].OriginalURL,
			UserID:      userID,
		}
		err := u.encoder.Encode(url)
		u.id++

		if err != nil {
			return errors.Wrap(err, "failed to add URLs")
		}
	}

	return nil
}

func (u *FileURLStore) IsAvailable(ctx context.Context) bool {
	return true
}

func (u *FileURLStore) GetUserURLs(ctx context.Context, userID domain.UserID) ([]domain.URLPair, error) {
	urls, _ := u.s.GetUserURLs(ctx, userID)
	return urls, nil
}

func (u *FileURLStore) DeleteUserURLs(ctx context.Context, shortURLs []string, userID domain.UserID) error {
	_ = u.s.DeleteUserURLs(ctx, shortURLs, userID)
	return nil
}
