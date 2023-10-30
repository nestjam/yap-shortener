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

func (s *inMemory) Get(shortUrl string) (string, error) {
	url, ok := s.m.Load(shortUrl)
	if !ok {
		return "", nil
	}
	return url.(string), nil
}

func (s *inMemory) Add(url string, shorten server.ShortenFunc) (string, error) {
	shortUrl := shorten(uuid.New().ID())
	s.m.Store(shortUrl, url)
	return shortUrl, nil
}
