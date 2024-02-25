package inmemory

import (
	"context"
	"errors"
	"sync"

	"github.com/nestjam/yap-shortener/internal/domain"
)

type InmemoryURLStore struct {
	m sync.Map
}

type urlRecord struct {
	originalURL string
	userID      domain.UserID
	isDeleted   bool
}

func New() *InmemoryURLStore {
	return &InmemoryURLStore{}
}

func (u *InmemoryURLStore) GetOriginalURL(ctx context.Context, shortURL string) (string, error) {
	value, ok := u.m.Load(shortURL)

	if !ok {
		return "", domain.ErrOriginalURLNotFound
	}

	rec, ok := value.(urlRecord)

	if !ok {
		return "", errors.New("failed type assertion")
	}

	if rec.isDeleted {
		return "", domain.ErrOriginalURLIsDeleted
	}

	return rec.originalURL, nil
}

func (u *InmemoryURLStore) AddURL(ctx context.Context, pair domain.URLPair, userID domain.UserID) error {
	if shortURL, ok := u.findShortURL(pair.OriginalURL); ok {
		return domain.NewOriginalURLExistsError(shortURL, nil)
	}

	rec := urlRecord{
		originalURL: pair.OriginalURL,
		userID:      userID,
	}
	u.m.Store(pair.ShortURL, rec)
	return nil
}

func (u *InmemoryURLStore) findShortURL(originalURL string) (string, bool) {
	shortURL := ""
	found := false

	u.m.Range(func(key, value any) bool {
		rec, ok := value.(urlRecord)

		if !ok {
			return true
		}

		if rec.originalURL == originalURL {
			found = true
			shortURL, _ = key.(string)
			return false
		}
		return true
	})

	return shortURL, found
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
		rec, ok := value.(urlRecord)

		if !ok || rec.userID != userID || rec.isDeleted {
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

func (u *InmemoryURLStore) DeleteUserURLs(ctx context.Context, shortURLs []string, userID domain.UserID) error {
	for _, shortURL := range shortURLs {
		value, ok := u.m.Load(shortURL)

		if !ok {
			continue
		}

		rec, ok := value.(urlRecord)

		if !ok {
			continue
		}

		if rec.userID == userID {
			rec.isDeleted = true
			_, _ = u.m.Swap(shortURL, rec)
		}
	}

	return nil
}
