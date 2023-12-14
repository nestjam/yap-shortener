package domain

import "context"

type URLPair struct {
	ShortURL    string
	OriginalURL string
}

type URLStore interface {
	GetOriginalURL(ctx context.Context, shortURL string) (string, error)
	AddURL(ctx context.Context, shortURL, originalURL string) error
	AddURLs(ctx context.Context, pairs []URLPair) error
	IsAvailable(ctx context.Context) bool
}
