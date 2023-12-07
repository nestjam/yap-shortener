package file

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/nestjam/yap-shortener/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	t.Run("get original url by short url", func(t *testing.T) {
		urls := []StoredURL{
			{ID: 1, ShortURL: "abc", OriginalURL: "http://mail.ru"},
			{ID: 2, ShortURL: "def", OriginalURL: "http://yandex.ru"},
		}
		rw := getReadWriter(t, urls)
		sut, _ := New(rw)

		for _, url := range urls {
			got, err := sut.Get(url.ShortURL)

			require.NoError(t, err)
			assert.Equal(t, url.OriginalURL, got)
		}
	})

	t.Run("original url is not stored", func(t *testing.T) {
		urls := []StoredURL{}
		rw := getReadWriter(t, urls)
		sut, _ := New(rw)

		_, err := sut.Get("abc")
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("invalid data", func(t *testing.T) {
		data := "invalid_data"
		rw := bytes.NewBuffer([]byte(data))
		_, err := New(rw)

		assert.Error(t, err)
	})
}

func TestAdd(t *testing.T) {
	t.Run("add new url", func(t *testing.T) {
		const (
			shortURL    = "abc"
			originalURL = "http://example.com"
		)
		want := StoredURL{ID: 0, ShortURL: shortURL, OriginalURL: originalURL}
		urls := []StoredURL{}
		rw := getReadWriter(t, urls)
		sut, _ := New(rw)

		sut.Add(shortURL, originalURL)

		got, err := sut.Get(shortURL)

		require.NoError(t, err)
		assert.Equal(t, want.OriginalURL, got)
		assertStoredURL(t, want, rw)
	})
}

func assertStoredURL(t *testing.T, want StoredURL, rw *bytes.Buffer) {
	t.Helper()
	decoder := json.NewDecoder(rw)
	var got StoredURL
	err := decoder.Decode(&got)

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func getReadWriter(t *testing.T, urls []StoredURL) *bytes.Buffer {
	t.Helper()
	var buf []byte

	for _, url := range urls {
		data, err := json.Marshal(url)

		require.NoError(t, err)

		buf = append(buf, data...)
	}

	return bytes.NewBuffer(buf)
}
