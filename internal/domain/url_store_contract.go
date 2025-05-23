package domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A URLStoreContract captures the expected behavior of a URL store
// in the form of tests that are run for a specific implementation of the store.
type URLStoreContract struct {
	NewURLStore func() (URLStore, func())
}

// Test задает набор тестов контракта хранилища сокращенных URL.
func (c URLStoreContract) Test(t *testing.T) {
	t.Run("add url", func(t *testing.T) {
		pair := URLPair{
			OriginalURL: "http://example.com",
			ShortURL:    "abc",
		}
		userID := NewUserID()
		sut, tearDown := c.NewURLStore()
		t.Cleanup(tearDown)

		err := sut.AddURL(context.Background(), pair, userID)
		assert.NoError(t, err)

		got, err := sut.GetOriginalURL(context.Background(), pair.ShortURL)

		require.NoError(t, err)
		assert.Equal(t, pair.OriginalURL, got)
	})

	t.Run("original url not found by short url", func(t *testing.T) {
		sut, tearDown := c.NewURLStore()
		t.Cleanup(tearDown)

		_, err := sut.GetOriginalURL(context.Background(), "123")
		assert.ErrorIs(t, err, ErrOriginalURLNotFound)
	})

	t.Run("get original url that is deleted", func(t *testing.T) {
		ctx := context.Background()
		pair := URLPair{
			OriginalURL: "http://example.com",
			ShortURL:    "abc",
		}
		userID := NewUserID()
		sut, tearDown := c.NewURLStore()
		t.Cleanup(tearDown)

		err := sut.AddURL(ctx, pair, userID)
		require.NoError(t, err)

		err = sut.DeleteUserURLs(ctx, []string{pair.ShortURL}, userID)
		require.NoError(t, err)

		_, err = sut.GetOriginalURL(ctx, pair.ShortURL)
		assert.ErrorIs(t, err, ErrOriginalURLIsDeleted)
	})

	t.Run("store is available", func(t *testing.T) {
		sut, tearDown := c.NewURLStore()
		t.Cleanup(tearDown)

		got := sut.IsAvailable(context.Background())
		assert.True(t, got)
	})

	t.Run("add batch of urls", func(t *testing.T) {
		ctx := context.Background()
		userID := NewUserID()
		pairs := []URLPair{
			{
				ShortURL:    "abc",
				OriginalURL: "http://example.com",
			},
			{
				ShortURL:    "123",
				OriginalURL: "http://yandex.ru",
			},
		}
		sut, tearDown := c.NewURLStore()
		t.Cleanup(tearDown)

		err := sut.AddURLs(ctx, pairs, userID)

		assert.NoError(t, err)

		for i := 0; i < len(pairs); i++ {
			got, err := sut.GetOriginalURL(ctx, pairs[i].ShortURL)
			require.NoError(t, err)
			assert.Equal(t, pairs[i].OriginalURL, got)
		}
	})

	t.Run("add same url twice", func(t *testing.T) {
		pair := URLPair{
			OriginalURL: "http://example.com",
			ShortURL:    "abc",
		}
		ctx := context.Background()
		userID := NewUserID()
		var want *OriginalURLExistsError
		sut, tearDown := c.NewURLStore()
		t.Cleanup(tearDown)

		err := sut.AddURL(ctx, pair, userID)

		require.NoError(t, err)

		got := sut.AddURL(ctx, pair, userID)

		assert.ErrorAs(t, got, &want)
		assert.Equal(t, pair.ShortURL, want.GetShortURL())
	})

	t.Run("get user urls", func(t *testing.T) {
		ctx := context.Background()
		sut, tearDown := c.NewURLStore()
		t.Cleanup(tearDown)

		userID := NewUserID()
		urls := []URLPair{
			{
				ShortURL:    "abc",
				OriginalURL: "http://example.com",
			},
			{
				ShortURL:    "123",
				OriginalURL: "http://yandex.ru",
			},
			{
				ShortURL:    "456",
				OriginalURL: "http://mail.ru",
			},
		}
		err := sut.AddURLs(ctx, urls[:2], userID)
		require.NoError(t, err)

		err = sut.AddURL(ctx, urls[2], userID)
		require.NoError(t, err)

		otherUserID := NewUserID()
		otherUrls := []URLPair{
			{
				ShortURL:    "xyz",
				OriginalURL: "http://google.com",
			},
		}
		err = sut.AddURLs(ctx, otherUrls, otherUserID)
		require.NoError(t, err)

		userURLs, err := sut.GetUserURLs(ctx, userID)

		assert.NoError(t, err)
		assert.ElementsMatch(t, urls, userURLs)
	})

	t.Run("delete requested user urls", func(t *testing.T) {
		ctx := context.Background()
		sut, tearDown := c.NewURLStore()
		t.Cleanup(tearDown)

		userID := NewUserID()
		urls := []URLPair{
			{
				ShortURL:    "abc",
				OriginalURL: "http://example.com",
			},
			{
				ShortURL:    "123",
				OriginalURL: "http://yandex.ru",
			},
			{
				ShortURL:    "456",
				OriginalURL: "http://mail.ru",
			},
		}
		err := sut.AddURLs(ctx, urls, userID)
		require.NoError(t, err)

		shortURLs := []string{urls[0].ShortURL, urls[1].ShortURL}
		err = sut.DeleteUserURLs(ctx, shortURLs, userID)

		assert.NoError(t, err)
		userURLs, err := sut.GetUserURLs(ctx, userID)
		require.NoError(t, err)

		assert.ElementsMatch(t, urls[2:], userURLs)
	})

	t.Run("other user attempts delete requested url", func(t *testing.T) {
		ctx := context.Background()
		sut, tearDown := c.NewURLStore()
		t.Cleanup(tearDown)

		userID := NewUserID()
		urls := []URLPair{
			{
				ShortURL:    "abc",
				OriginalURL: "http://example.com",
			},
		}
		err := sut.AddURLs(ctx, urls, userID)
		require.NoError(t, err)

		shortURLs := []string{urls[0].ShortURL}
		otherUserID := NewUserID()
		err = sut.DeleteUserURLs(ctx, shortURLs, otherUserID)

		assert.NoError(t, err)
		userURLs, err := sut.GetUserURLs(ctx, userID)
		require.NoError(t, err)

		assert.ElementsMatch(t, urls, userURLs)
	})

	t.Run("delete requested url that is not stored", func(t *testing.T) {
		ctx := context.Background()
		sut, tearDown := c.NewURLStore()
		t.Cleanup(tearDown)

		userID := NewUserID()
		urls := []URLPair{
			{
				ShortURL:    "abc",
				OriginalURL: "http://example.com",
			},
		}
		err := sut.AddURLs(ctx, urls, userID)
		require.NoError(t, err)

		shortURLs := []string{"123", urls[0].ShortURL}
		err = sut.DeleteUserURLs(ctx, shortURLs, userID)

		assert.NoError(t, err)
		userURLs, err := sut.GetUserURLs(ctx, userID)
		require.NoError(t, err)

		assert.Empty(t, userURLs)
	})
}
