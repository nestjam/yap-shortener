package store

import (
	"sync"

	"github.com/nestjam/yap-shortener/internal/model"
)

type inMemory struct {
	m sync.Map
}

func NewInMemory() *inMemory {
	return &inMemory{}
}

func (s *inMemory) Get(shortURL string) (string, error) {
	url, ok := s.m.Load(shortURL)
	if !ok {
		return "", model.ErrNotFound
	}
	return url.(string), nil
}

func (s *inMemory) Add(shortURL, url string) {
	s.m.Store(shortURL, url)
}
