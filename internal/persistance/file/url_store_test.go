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

func TestFileURLStore(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running test.")
	}
	domain.URLStoreContract{
		NewURLStore: func() (domain.URLStore, func()) {
			t.Helper()
			f, err := os.CreateTemp(os.TempDir(), "*")

			require.NoError(t, err)

			store, err := New(context.Background(), f)

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
			urls = []StoredURL{
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
			rw = getReadWriter(t, urls)
		)

		_, err := New(context.Background(), rw)

		var want *domain.OriginalURLExistsError
		require.ErrorAs(t, err, &want)
	})
}

func TestGet(t *testing.T) {
	t.Run("invalid data", func(t *testing.T) {
		data := "invalid_data"
		rw := bytes.NewBuffer([]byte(data))
		_, err := New(context.Background(), rw)

		assert.Error(t, err)
	})
}

func TestAddURL(t *testing.T) {
	t.Run("write url to writer", func(t *testing.T) {
		const (
			shortURL    = "abc"
			originalURL = "http://example.com"
		)
		ctx := context.Background()
		userID := domain.NewUserID()
		want := StoredURL{
			ID:          0,
			ShortURL:    shortURL,
			OriginalURL: originalURL,
			UserID:      userID,
		}
		pair := domain.URLPair{
			ShortURL:    shortURL,
			OriginalURL: originalURL,
		}
		urls := []StoredURL{}
		rw := getReadWriter(t, urls)
		sut, _ := New(ctx, rw)
		err := sut.AddURL(ctx, pair, userID)
		require.NoError(t, err)

		assertStoredURL(t, want, rw)
	})
}

func TestAddURLs(t *testing.T) {
	t.Run("write batch of urls to writer", func(t *testing.T) {
		var (
			userID = domain.NewUserID()
			urls   = []domain.URLPair{
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
					UserID:      userID,
				},
				{
					ID:          1,
					ShortURL:    urls[1].ShortURL,
					OriginalURL: urls[1].OriginalURL,
					UserID:      userID,
				},
			}
			stored = []StoredURL{}
			rw     = getReadWriter(t, stored)
			sut, _ = New(context.Background(), rw)
		)

		err := sut.AddURLs(context.Background(), urls, userID)

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
