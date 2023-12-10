package file

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/nestjam/yap-shortener/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestURLStore(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running test.")
	}
	domain.URLStoreContract{
		NewURLStore: func() (domain.URLStore, func()) {
			t.Helper()
			f, err := os.CreateTemp(os.TempDir(), "*")

			require.NoError(t, err)

			store, err := New(f)

			require.NoError(t, err)

			return store, func() {
				_ = f.Close()
				_ = os.Remove(f.Name())
			}
		},
	}.Test(t)
}

func TestNew(t *testing.T) {
	t.Run("data contains records with same original url", func(t *testing.T) {
		const originalURL = "http://example.com"
		var (
			urls   = []StoredURL{
				{
					ID:          0,
					ShortURL:    "abc",
					OriginalURL: originalURL,
				},
				{
					ID:          1,
					ShortURL:    "123",
					OriginalURL: originalURL,
				},
			}
			rw     = getReadWriter(t, urls)
		)

		_, err := New(rw)

		var want *domain.OriginalURLExistsError
		require.ErrorAs(t, err, &want)
	})
}

func TestGet(t *testing.T) {
	t.Run("invalid data", func(t *testing.T) {
		data := "invalid_data"
		rw := bytes.NewBuffer([]byte(data))
		_, err := New(rw)

		assert.Error(t, err)
	})
}

func TestAdd(t *testing.T) {
	t.Run("write url to writer", func(t *testing.T) {
		const (
			shortURL    = "abc"
			originalURL = "http://example.com"
		)
		var (
			want   = StoredURL{ID: 0, ShortURL: shortURL, OriginalURL: originalURL}
			urls   = []StoredURL{}
			rw     = getReadWriter(t, urls)
			sut, _ = New(rw)
		)

		err := sut.Add(context.Background(), shortURL, originalURL)
		require.NoError(t, err)

		assertStoredURL(t, want, rw)
	})
}

func TestAddBatch(t *testing.T) {
	t.Run("write batch of urls to writer", func(t *testing.T) {
		var (
			urls = []domain.URLPair{
				{
					ShortURL:    "abc",
					OriginalURL: "http://example.com",
				},
				{
					ShortURL:    "123",
					OriginalURL: "http://example2.com",
				},
			}
			want = []StoredURL{
				{
					ID:          0,
					ShortURL:    urls[0].ShortURL,
					OriginalURL: urls[0].OriginalURL,
				},
				{
					ID:          1,
					ShortURL:    urls[1].ShortURL,
					OriginalURL: urls[1].OriginalURL,
				},
			}
			stored = []StoredURL{}
			rw     = getReadWriter(t, stored)
			sut, _ = New(rw)
		)

		err := sut.AddBatch(context.Background(), urls)

		require.NoError(t, err)
		assertStoredURLs(t, want, rw)
	})
}

func assertStoredURLs(t *testing.T, wantURLs []StoredURL, rw *bytes.Buffer) {
	t.Helper()

	dec := json.NewDecoder(rw)

	for _, want := range wantURLs {
		var got StoredURL
		err := dec.Decode(&got)
		require.NoError(t, err)

		assert.Equal(t, want, got)
	}
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
