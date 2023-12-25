package server

import (
	"context"
	"testing"

	"github.com/nestjam/yap-shortener/internal/domain"
	"github.com/nestjam/yap-shortener/internal/persistance/inmemory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveUserURLs(t *testing.T) {
	t.Run("delete urls from single channel", func(t *testing.T) {
		ctx := context.Background()
		store := inmemory.New()
		userID := domain.NewUserID()
		doneCh := make(chan struct{})
		defer close(doneCh)
		sut := NewURLRemover(ctx, doneCh, store)
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
		sut.Delete(shortURLs, userID)

		go func() {
			<-doneCh
			userURLs, err := store.GetUserURLs(ctx, userID)
			require.NoError(t, err)
			assert.Empty(t, userURLs)
		}()
	})
}
