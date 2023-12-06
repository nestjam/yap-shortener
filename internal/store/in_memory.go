package store

import (
	"sync"

	"github.com/nestjam/yap-shortener/internal/model"
)

type InMemoryStorage struct {
	m sync.Map
}

func NewInMemory() *InMemoryStorage {
	return &InMemoryStorage{}
}

func (s *InMemoryStorage) Get(shortURL string) (string, error) {
	url, ok := s.m.Load(shortURL)
	if !ok {
		return "", model.ErrNotFound
	}
	return url.(string), nil
}

func (s *InMemoryStorage) Add(shortURL, url string) {
	s.m.Store(shortURL, url)
}

func (s *InMemoryStorage) IsAvailable() bool {
	return true
}
