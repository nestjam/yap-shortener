package inmemory

import (
	"sync"

	"github.com/nestjam/yap-shortener/internal/domain"
)

type URLStore struct {
	m sync.Map
}

func New() *URLStore {
	return &URLStore{}
}

func (u *URLStore) Get(shortURL string) (string, error) {
	url, ok := u.m.Load(shortURL)
	if !ok {
		return "", domain.ErrURLNotFound
	}
	return url.(string), nil
}

func (u *URLStore) Add(shortURL, url string) error {
	u.m.Store(shortURL, url)
	return nil
}

func (u *URLStore) AddBatch(pairs []domain.URLPair) error {
	for _, p := range pairs {
		u.m.Store(p.ShortURL, p.OriginalURL)
	}
	return nil
}

func (u *URLStore) IsAvailable() bool {
	return true
}
