package domain

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	ErrURLNotFound = errors.New("not found")
)

type URLPair struct {
	ShortURL    string
	OriginalURL string
}

type URLStore interface {
	Get(ctx context.Context, shortURL string) (string, error)
	Add(ctx context.Context, shortURL, originalURL string) error
	AddBatch(ctx context.Context, pairs []URLPair) error
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

		err := sut.Add(context.Background(), shortURL, originalURL)

		assert.NoError(t, err)

		got, err := sut.Get(context.Background(), shortURL)

		require.NoError(t, err)
		assert.Equal(t, originalURL, got)
	})

	t.Run("original url not found by short url", func(t *testing.T) {
		sut, tearDown := c.NewURLStore()
		t.Cleanup(tearDown)

		_, err := sut.Get(context.Background(), "123")
		assert.ErrorIs(t, err, ErrURLNotFound)
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

		err := sut.AddBatch(context.Background(), pairs)

		assert.NoError(t, err)

		for i := 0; i < len(pairs); i++ {
			got, err := sut.Get(context.Background(), pairs[i].ShortURL)
			require.NoError(t, err)
			assert.Equal(t, pairs[i].OriginalURL, got)
		}
	})
}
