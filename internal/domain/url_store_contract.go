package domain

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	ErrURLNotFound = errors.New("not found")
)

type URLStore interface {
	Get(shortURL string) (string, error)
	Add(shortURL, url string)
	IsAvailable() bool
}

type URLStoreContract struct {
	NewURLStore func() (URLStore, func())
}

func (c URLStoreContract) Test(t *testing.T) {
	t.Run("add new url", func(t *testing.T) {
		const (
			shortURL    = "abc"
			originalURL = "http://example.com"
		)
		sut, tearDown := c.NewURLStore()
		t.Cleanup(tearDown)

		sut.Add(shortURL, originalURL)

		got, err := sut.Get(shortURL)
		require.NoError(t, err)
		assert.Equal(t, originalURL, got)
	})

	t.Run("original url not found by short url", func(t *testing.T) {
		sut, tearDown := c.NewURLStore()
		t.Cleanup(tearDown)

		_, err := sut.Get("123")
		assert.ErrorIs(t, err, ErrURLNotFound)
	})

	t.Run("store is available", func(t *testing.T) {
		sut, tearDown := c.NewURLStore()
		t.Cleanup(tearDown)

		got := sut.IsAvailable()
		assert.True(t, got)
	})
}
