package server

import (
	"context"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nestjam/yap-shortener/internal/domain"
	"github.com/nestjam/yap-shortener/internal/persistance/inmemory"
)

func BenchmarkURLShortener(b *testing.B) {
	b.Run("with in memory store", func(b *testing.B) {
		URLShortenerTest{
			CreateDependencies: func() (domain.URLStore, Cleanup) {
				return inmemory.New(), func() {
				}
			},
		}.Benchmark(b)
	})
}

func (u URLShortenerTest) Benchmark(b *testing.B) {
	b.Run("redirect to original url", func(b *testing.B) {
		const shortURL = "EwHXdJfB"
		userID := domain.NewUserID()
		pair := domain.URLPair{
			ShortURL:    shortURL,
			OriginalURL: testURL,
		}
		urlStore, cleanup := u.CreateDependencies()
		b.Cleanup(cleanup)
		err := urlStore.AddURL(context.Background(), pair, userID)
		require.NoError(b, err)
		sut := New(urlStore, baseURL)
		request := newGetRequest(shortURL)
		response := httptest.NewRecorder()

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			sut.ServeHTTP(response, request)
		}
	})

	b.Run("shorten url", func(b *testing.B) {
		urlStore, cleanup := u.CreateDependencies()
		b.Cleanup(cleanup)
		sut := New(urlStore, baseURL)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			b.StopTimer()
			request := newShortenRequest(testURL + strconv.Itoa(i))
			response := httptest.NewRecorder()
			b.StartTimer()

			sut.ServeHTTP(response, request)
		}
	})
}
