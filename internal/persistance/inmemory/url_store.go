package inmemory

import (
	"context"
	"sync"

	"github.com/nestjam/yap-shortener/internal/domain"
)

type InmemoryURLStore struct {
	m sync.Map
}

func New() *InmemoryURLStore {
	return &InmemoryURLStore{}
}

func (u *InmemoryURLStore) GetOriginalURL(ctx context.Context, shortURL string) (string, error) {
	url, ok := u.m.Load(shortURL)
	if !ok {
		return "", domain.ErrOriginalURLNotFound
	}
	return url.(string), nil
}

func (u *InmemoryURLStore) AddURL(ctx context.Context, shortURL, originalURL string) error {
	if shortURL, ok := u.findShortURL(originalURL); ok {
		return domain.NewOriginalURLExistsError(shortURL, nil)
	}

	u.m.Store(shortURL, originalURL)
	return nil
}

func (u *InmemoryURLStore) findShortURL(originalURL string) (string, bool) {
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

func (u *InmemoryURLStore) AddURLs(ctx context.Context, pairs []domain.URLPair) error {
	for _, p := range pairs {
		u.m.Store(p.ShortURL, p.OriginalURL)
	}
	return nil
}

func (u *InmemoryURLStore) IsAvailable(ctx context.Context) bool {
	return true
}
