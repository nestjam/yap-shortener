package domain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

type URLStoreContract struct {
	NewURLStore func() (URLStore, func())
}

func (c URLStoreContract) Test(t *testing.T) {
	t.Run("add url", func(t *testing.T) {
		const (
			shortURL    = "abc"
			originalURL = "http://example.com"
		)
		sut, tearDown := c.NewURLStore()
		t.Cleanup(tearDown)

		err := sut.AddURL(context.Background(), shortURL, originalURL)

		assert.NoError(t, err)

		got, err := sut.GetOriginalURL(context.Background(), shortURL)

		require.NoError(t, err)
		assert.Equal(t, originalURL, got)
	})

	t.Run("original url not found by short url", func(t *testing.T) {
		sut, tearDown := c.NewURLStore()
		t.Cleanup(tearDown)

		_, err := sut.GetOriginalURL(context.Background(), "123")
		assert.ErrorIs(t, err, ErrOriginalURLNotFound)
	})

	t.Run("store is available", func(t *testing.T) {
		sut, tearDown := c.NewURLStore()
		t.Cleanup(tearDown)

		got := sut.IsAvailable(context.Background())
		assert.True(t, got)
	})

	t.Run("add batch of urls", func(t *testing.T) {
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

		err := sut.AddURLs(context.Background(), pairs)

		assert.NoError(t, err)

		for i := 0; i < len(pairs); i++ {
			got, err := sut.GetOriginalURL(context.Background(), pairs[i].ShortURL)
			require.NoError(t, err)
			assert.Equal(t, pairs[i].OriginalURL, got)
		}
	})

	t.Run("add same url twice", func(t *testing.T) {
		const (
			shortURL    = "abc"
			originalURL = "http://example.com"
		)
		ctx := context.Background()
		var want *OriginalURLExistsError
		sut, tearDown := c.NewURLStore()
		t.Cleanup(tearDown)

		err := sut.AddURL(ctx, shortURL, originalURL)

		require.NoError(t, err)

		got := sut.AddURL(ctx, shortURL, originalURL)

		assert.ErrorAs(t, got, &want)
		assert.Equal(t, shortURL, want.GetShortURL())
	})
}
