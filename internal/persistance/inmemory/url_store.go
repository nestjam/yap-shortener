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

func (s *URLStore) Get(shortURL string) (string, error) {
	url, ok := s.m.Load(shortURL)
	if !ok {
		return "", domain.ErrURLNotFound
	}
	return url.(string), nil
}

func (s *URLStore) Add(shortURL, url string) {
	s.m.Store(shortURL, url)
}

func (s *URLStore) IsAvailable() bool {
	return true
}
