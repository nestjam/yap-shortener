package store

import (
	"sync"

	"github.com/google/uuid"
	"github.com/nestjam/yap-shortener/internal/server"
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
		return "", nil
	}
	return url.(string), nil
}

func (s *inMemory) Add(url string, shorten server.ShortenFunc) (string, error) {
	shortURL := shorten(uuid.New().ID())
	s.m.Store(shortURL, url)
	return shortURL, nil
}
