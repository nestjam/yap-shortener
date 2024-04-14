package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/nestjam/yap-shortener/internal/domain"
	"github.com/nestjam/yap-shortener/internal/shortener"
	"github.com/pkg/errors"
)

// Ошибки, связанные с сокращением ссылки.
var (
	ErrURLIsEmpty = errors.New("url is empty") // исходный URL удален
)

// ShortenerService выполняет сокращение и получение полной ссылки, удаление сокращенных ссылок.
type ShortenerService struct {
	store      domain.URLStore
	urlRemover *URLRemover
}

// New создает сервис сокращения ссылок.
func New(store domain.URLStore) *ShortenerService {
	return &ShortenerService{
		store: store,
	}
}

// SetURLRemover задает компонент, удаляющий ссылки.
func (s *ShortenerService) SetURLRemover(remover *URLRemover) {
	s.urlRemover = remover
}

// GetOriginalURL возвращает исходную ссылку по сокращенной.
func (s *ShortenerService) GetOriginalURL(ctx context.Context, key string) (string, error) {
	const op = "get original url"

	url, err := s.store.GetOriginalURL(ctx, key)
	if err != nil {
		return "", errors.Wrap(err, op)
	}

	return url, nil
}

// ShortenURL сокращает исходную ссылку.
func (s *ShortenerService) ShortenURL(ctx context.Context, url string, user domain.UserID) (string, error) {
	const op = "shorten url"

	if len(url) == 0 {
		return "", ErrURLIsEmpty
	}

	key := shortener.Shorten(uuid.New().ID())
	pair := domain.URLPair{
		ShortURL:    key,
		OriginalURL: url,
	}

	err := s.store.AddURL(ctx, pair, user)
	if err != nil {
		return "", errors.Wrap(err, op)
	}

	return key, nil
}

// ShortenURLs сокращает набор исходных ссылок.
func (s *ShortenerService) ShortenURLs(ctx context.Context, urls []string, u domain.UserID) ([]domain.URLPair, error) {
	const op = "shorten urls"

	urlPairs := make([]domain.URLPair, len(urls))
	for i := 0; i < len(urls); i++ {
		url := urls[i]

		if len(url) == 0 {
			return nil, ErrURLIsEmpty
		}

		urlPairs[i] = domain.URLPair{
			ShortURL:    shortener.Shorten(uuid.New().ID()),
			OriginalURL: url,
		}
	}

	err := s.store.AddURLs(ctx, urlPairs, u)
	if err != nil {
		return nil, errors.Wrap(err, op)
	}

	return urlPairs, nil
}

// GetUserURLs возвращает набор пар исходных и сокращенных ссылок указанного пользователя.
func (s *ShortenerService) GetUserURLs(ctx context.Context, user domain.UserID) ([]domain.URLPair, error) {
	const op = "get user urls"

	urlPairs, err := s.store.GetUserURLs(ctx, user)
	if err != nil {
		return nil, errors.Wrap(err, op)
	}

	return urlPairs, nil
}

// DeleteUserURLs удаляет сокращенные ссылки указанного пользователя.
func (s *ShortenerService) DeleteUserURLs(ctx context.Context, keys []string, user domain.UserID) error {
	const op = "delete user urls"

	var err error
	if s.urlRemover != nil {
		err = s.urlRemover.DeleteURLs(keys, user)
	} else {
		err = s.store.DeleteUserURLs(ctx, keys, user)
	}

	if err != nil {
		return errors.Wrap(err, op)
	}

	return nil
}

// IsAvailable возвращает true, если сервис доступен.
func (s *ShortenerService) IsAvailable(ctx context.Context) bool {
	return s.store.IsAvailable(ctx)
}
