package domain

import "context"

type URLPair struct {
	ShortURL    string
	OriginalURL string
}

type URLStore interface {
	GetOriginalURL(ctx context.Context, shortURL string) (string, error)
	AddURL(ctx context.Context, pair URLPair, userID UserID) error
	AddURLs(ctx context.Context, pairs []URLPair, userID UserID) error
	GetUserURLs(ctx context.Context, userID UserID) ([]URLPair, error)
	DeleteUserURLs(ctx context.Context, shortURLs []string, userID UserID) error
	IsAvailable(ctx context.Context) bool
}
