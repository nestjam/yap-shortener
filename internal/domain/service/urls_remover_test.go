package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/nestjam/yap-shortener/internal/domain"
	"github.com/nestjam/yap-shortener/internal/persistance/inmemory"
)

func TestDeleteURLs(t *testing.T) {
	t.Run("delete urls", func(t *testing.T) {
		ctx := context.Background()
		store := inmemory.New()
		userID := domain.NewUserID()
		doneCh := make(chan struct{})
		defer close(doneCh)
		sut := NewURLRemover(ctx, doneCh, store, zap.NewNop())
		urls := []domain.URLPair{
			{
				OriginalURL: "http://yandex.ru",
				ShortURL:    "123",
			},
			{
				OriginalURL: "http://mail.ru",
				ShortURL:    "abc",
			},
		}
		err := store.AddURLs(ctx, urls, userID)
		require.NoError(t, err)

		shortURLs := []string{
			urls[0].ShortURL,
			urls[1].ShortURL,
		}
		err = sut.DeleteURLs(shortURLs, userID)
		require.NoError(t, err)

		time.Sleep(10 * time.Millisecond)

		userURLs, err := store.GetUserURLs(ctx, userID)
		require.NoError(t, err)
		assert.Empty(t, userURLs)
	})

	t.Run("error on delete urls after closing", func(t *testing.T) {
		ctx := context.Background()
		store := inmemory.New()
		userID := domain.NewUserID()
		doneCh := make(chan struct{})
		sut := NewURLRemover(ctx, doneCh, store, zap.NewNop())
		close(doneCh)

		err := sut.DeleteURLs([]string{"abc"}, userID)
		assert.NotNil(t, err)
	})
}
