package inmemory

import (
	"context"
	"sync"

	"github.com/nestjam/yap-shortener/internal/domain"
)

type URLStore struct {
	m sync.Map
}

func New() *URLStore {
	return &URLStore{}
}

func (u *URLStore) Get(ctx context.Context, shortURL string) (string, error) {
	url, ok := u.m.Load(shortURL)
	if !ok {
		return "", domain.ErrOriginalURLNotFound
	}
	return url.(string), nil
}

func (u *URLStore) Add(ctx context.Context, shortURL, originalURL string) error {
	if shortURL, ok := u.findShortURL(originalURL); ok {
		return domain.NewOriginalURLExistsError(shortURL, nil)
	}

	u.m.Store(shortURL, originalURL)
	return nil
}

func (u *URLStore) findShortURL(originalURL string) (string, bool) {
	shortURL := ""
	ok := false

	u.m.Range(func(key, value any) bool {
		if value.(string) == originalURL {
			ok = true
			shortURL, _ = key.(string)
			return false
		}
		return true
	})

	return shortURL, ok
}

func (u *URLStore) AddBatch(ctx context.Context, pairs []domain.URLPair) error {
	for _, p := range pairs {
		u.m.Store(p.ShortURL, p.OriginalURL)
	}
	return nil
}

func (u *URLStore) IsAvailable(ctx context.Context) bool {
	return true
}
