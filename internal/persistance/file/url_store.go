package file

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/nestjam/yap-shortener/internal/domain"
	"github.com/pkg/errors"
)

// FileURLStore реализует хранилище ссылок на основе файла.
type FileURLStore struct {
	encoder *json.Encoder
	m       map[string]StoredURL
	mu      sync.Mutex
}

// StoredURL описывает данные сокращенной ссылки.
type StoredURL struct {
	ShortURL    string        `json:"short_url"`    // сокращенный URL
	OriginalURL string        `json:"original_url"` // исходный URL
	UserID      domain.UserID `json:"user_id"`      // идентификатор пользователя
	IsDeleted   bool          `json:"is_deleted"`   // признак удаленной ссылки
}

// New создает экземпляр файлового хранилища.
func New(ctx context.Context, rw io.ReadWriter) (*FileURLStore, error) {
	const op = "new file storage"
	m, err := readURLs(rw)

	if err != nil {
		return nil, errors.Wrap(err, op)
	}

	store := FileURLStore{
		encoder: json.NewEncoder(rw),
		m:       m,
	}
	return &store, nil
}

func readURLs(rw io.ReadWriter) (map[string]StoredURL, error) {
	dec := json.NewDecoder(rw)
	m := make(map[string]StoredURL)

	for dec.More() {
		var rec StoredURL
		err := dec.Decode(&rec)

		if err != nil {
			return nil, fmt.Errorf("get URLs: %w", err)
		}

		if _, ok := m[rec.ShortURL]; !ok {
			if shortURL, ok := findShortURL(m, rec.OriginalURL); ok {
				return nil, domain.NewOriginalURLExistsError(shortURL, nil)
			}
		}

		m[rec.ShortURL] = rec
	}

	return m, nil
}

// GetOriginalURL возвращает исходный URL для сокращенного URL или ошибку.
func (u *FileURLStore) GetOriginalURL(ctx context.Context, shortURL string) (string, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	rec, ok := u.m[shortURL]

	if !ok {
		return "", domain.ErrOriginalURLNotFound
	}

	if rec.IsDeleted {
		return "", domain.ErrOriginalURLIsDeleted
	}

	return rec.OriginalURL, nil
}

// AddURL добавляет в хранилище пару исходный и сокращенный URL.
func (u *FileURLStore) AddURL(ctx context.Context, pair domain.URLPair, userID domain.UserID) error {
	const op = "add URL"
	u.mu.Lock()
	defer u.mu.Unlock()

	if shortURL, ok := findShortURL(u.m, pair.OriginalURL); ok {
		return domain.NewOriginalURLExistsError(shortURL, nil)
	}

	rec := StoredURL{
		ShortURL:    pair.ShortURL,
		OriginalURL: pair.OriginalURL,
		UserID:      userID,
	}
	u.m[rec.ShortURL] = rec

	err := u.encoder.Encode(rec)

	if err != nil {
		return errors.Wrap(err, op)
	}

	return nil
}

func findShortURL(m map[string]StoredURL, originalURL string) (string, bool) {
	for k, v := range m {
		if v.OriginalURL == originalURL {
			return k, true
		}
	}

	return "", false
}

// AddURLs добавляет в хранилище коллекцию пар исходного и сокращенного URL.
func (u *FileURLStore) AddURLs(ctx context.Context, pairs []domain.URLPair, userID domain.UserID) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	for _, url := range pairs {
		rec := StoredURL{
			ShortURL:    url.ShortURL,
			OriginalURL: url.OriginalURL,
			UserID:      userID,
		}
		u.m[rec.ShortURL] = rec

		err := u.encoder.Encode(rec)

		if err != nil {
			return errors.Wrap(err, "failed to add URLs")
		}
	}
	return nil
}

// IsAvailable позволяет проверить доступность хранилща.
func (u *FileURLStore) IsAvailable(ctx context.Context) bool {
	return true
}

// GetUserURLs возвращает коллекцию пар исходного и сокращенного URL, которые были добавлены указанным пользователем.
func (u *FileURLStore) GetUserURLs(ctx context.Context, userID domain.UserID) ([]domain.URLPair, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	userURLs := []domain.URLPair{}

	for k, v := range u.m {
		if v.UserID != userID || v.IsDeleted {
			continue
		}

		url := domain.URLPair{
			ShortURL:    k,
			OriginalURL: v.OriginalURL,
		}
		userURLs = append(userURLs, url)
	}

	return userURLs, nil
}

// DeleteUserURLs удаляет из хранилища коллекцию пар исходного и сокращенного URL, которые были добавлены указанным пользователем.
func (u *FileURLStore) DeleteUserURLs(ctx context.Context, shortURLs []string, userID domain.UserID) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	for _, shortURL := range shortURLs {
		rec, ok := u.m[shortURL]

		if !ok {
			continue
		}

		if rec.UserID == userID {
			rec.IsDeleted = true
			u.m[shortURL] = rec

			err := u.encoder.Encode(rec)

			if err != nil {
				return errors.Wrap(err, "failed write url")
			}
		}
	}

	return nil
}
