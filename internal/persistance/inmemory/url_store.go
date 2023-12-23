package inmemory

import (
	"context"
	"sync"

	"github.com/nestjam/yap-shortener/internal/domain"
)

type InmemoryURLStore struct {
	m sync.Map
}

type urlRecord struct {
	originalURL string
	userID      domain.UserID
}

func New() *InmemoryURLStore {
	return &InmemoryURLStore{}
}

func (u *InmemoryURLStore) GetOriginalURL(ctx context.Context, shortURL string) (string, error) {
	rec, ok := u.m.Load(shortURL)
	if !ok {
		return "", domain.ErrOriginalURLNotFound
	}
	return (rec.(urlRecord)).originalURL, nil
}

func (u *InmemoryURLStore) AddURL(ctx context.Context, pair domain.URLPair, userID domain.UserID) error {
	if shortURL, ok := u.findShortURL(pair.OriginalURL); ok {
		return domain.NewOriginalURLExistsError(shortURL, nil)
	}

	rec := urlRecord{
		originalURL: pair.OriginalURL,
		userID: userID,
	}
	u.m.Store(pair.ShortURL, rec)
	return nil
}

func (u *InmemoryURLStore) findShortURL(originalURL string) (string, bool) {
	shortURL := ""
	ok := false

	u.m.Range(func(key, value any) bool {
		rec := value.(urlRecord)
		if rec.originalURL == originalURL {
			ok = true
			shortURL, _ = key.(string)
			return false
		}
		return true
	})

	return shortURL, ok
}

func (u *InmemoryURLStore) AddURLs(ctx context.Context, urls []domain.URLPair, userID domain.UserID) error {
	for _, url := range urls {
		rec := urlRecord{
			originalURL: url.OriginalURL,
			userID:      userID,
		}
		u.m.Store(url.ShortURL, rec)
	}
	return nil
}

func (u *InmemoryURLStore) IsAvailable(ctx context.Context) bool {
	return true
}

func (u *InmemoryURLStore) GetUserURLs(ctx context.Context, userID domain.UserID) ([]domain.URLPair, error) {
	var userURLs []domain.URLPair

	u.m.Range(func(key, value any) bool {
		rec := value.(urlRecord)

		if rec.userID != userID {
			return true
		}

		url := domain.URLPair{
			ShortURL:    key.(string),
			OriginalURL: rec.originalURL,
		}
		userURLs = append(userURLs, url)
		return true
	})

	return userURLs, nil
}
