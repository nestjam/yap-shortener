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
		return "", domain.ErrURLNotFound
	}
	return url.(string), nil
}

func (u *URLStore) Add(ctx context.Context, shortURL, url string) error {
	u.m.Store(shortURL, url)
	return nil
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
