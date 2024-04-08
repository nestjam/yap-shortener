package grpc

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/nestjam/yap-shortener/internal/domain"
	"github.com/nestjam/yap-shortener/internal/persistance/inmemory"
	pb "github.com/nestjam/yap-shortener/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	customctx "github.com/nestjam/yap-shortener/internal/context"
)

const (
	testURL = "https://practicum.yandex.ru/"
	baseURL = "http://localhost:8080"
)

func TestURLShortener(t *testing.T) {
	t.Run("with in memory store", func(t *testing.T) {
		URLShortenerTest{
			CreateDependencies: func() (domain.URLStore, Cleanup) {
				return inmemory.New(), func() {
				}
			},
		}.Test(t)
	})
}

type Cleanup func()

type URLShortenerTest struct {
	CreateDependencies func() (domain.URLStore, Cleanup)
}

func (u URLShortenerTest) Test(t *testing.T) {
	t.Run("pinging service", func(t *testing.T) {
		t.Run("service is avaiable", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			ctx := context.Background()
			request := &pb.PingRequest{}
			resp, err := sut.Ping(ctx, request)

			require.NoError(t, err)
			assert.Equal(t, true, resp.Result)
		})

		t.Run("service is not avaiable", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			unavailableURLStore := domain.NewURLStoreDelegate(urlStore)
			unavailableURLStore.IsAvailableFunc = func(ctx context.Context) bool {
				return false
			}
			sut := New(unavailableURLStore, baseURL)
			ctx := context.Background()
			request := &pb.PingRequest{}
			resp, err := sut.Ping(ctx, request)

			require.NoError(t, err)
			assert.Equal(t, false, resp.Result)
		})
	})

	t.Run("getting original url", func(t *testing.T) {
		t.Run("redirect to original url", func(t *testing.T) {
			const key = "EwHXdJfB"
			userID := domain.NewUserID()
			pair := domain.URLPair{
				ShortURL:    key,
				OriginalURL: testURL,
			}
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			err := urlStore.AddURL(context.Background(), pair, userID)
			require.NoError(t, err)
			sut := New(urlStore, baseURL)
			ctx := context.Background()
			request := &pb.GetOriginalURLRequest{Key: key}

			resp, err := sut.GetOriginalURL(ctx, request)

			require.NoError(t, err)
			assert.Equal(t, testURL, resp.OriginalURL)
		})

		t.Run("url not found", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			ctx := context.Background()
			request := &pb.GetOriginalURLRequest{Key: "EwHXdJfB"}

			_, err := sut.GetOriginalURL(ctx, request)

			require.Error(t, err)
			assertCode(t, codes.NotFound, err)
		})

		t.Run("url is deleted", func(t *testing.T) {
			const key = "EwHXdJfB"
			ctx := context.Background()
			userID := domain.NewUserID()
			pair := domain.URLPair{
				ShortURL:    key,
				OriginalURL: testURL,
			}
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			err := urlStore.AddURL(ctx, pair, userID)
			require.NoError(t, err)

			err = urlStore.DeleteUserURLs(ctx, []string{pair.ShortURL}, userID)
			require.NoError(t, err)

			sut := New(urlStore, baseURL)
			request := &pb.GetOriginalURLRequest{Key: key}

			_, err = sut.GetOriginalURL(ctx, request)

			require.Error(t, err)
			assertCode(t, codes.NotFound, err)
		})
	})

	t.Run("shortening url", func(t *testing.T) {
		t.Run("shorten url", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			ctx := context.Background()
			ctx = setNewUser(ctx)
			request := &pb.ShortenURLRequest{
				URL: testURL,
			}

			resp, err := sut.ShortenURL(ctx, request)

			require.NoError(t, err)
			assertRedirectURL(t, resp.ShortenedURL, urlStore)
		})

		t.Run("shorten same url twice", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			ctx := context.Background()
			ctx = setNewUser(ctx)

			//1
			request := &pb.ShortenURLRequest{
				URL: testURL,
			}

			resp, err := sut.ShortenURL(ctx, request)

			require.NoError(t, err)
			assertRedirectURL(t, resp.ShortenedURL, urlStore)

			//2
			resp, err = sut.ShortenURL(ctx, request)

			require.Error(t, err)
			assertCode(t, codes.AlreadyExists, err)
			assertRedirectURL(t, resp.ShortenedURL, urlStore)
		})

		t.Run("url is empty", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			ctx := context.Background()
			ctx = setNewUser(ctx)
			request := &pb.ShortenURLRequest{
				URL: "",
			}

			_, err := sut.ShortenURL(ctx, request)

			assert.Error(t, err)
			assertCode(t, codes.InvalidArgument, err)
		})

		t.Run("failed to store url", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			failingURLStore := domain.NewURLStoreDelegate(urlStore)
			failingURLStore.AddURLFunc = func(ctx context.Context, pair domain.URLPair, userID domain.UserID) error {
				return errors.New("failed to add url")
			}
			sut := New(failingURLStore, baseURL)
			ctx := context.Background()
			ctx = setNewUser(ctx)
			request := &pb.ShortenURLRequest{
				URL: testURL,
			}

			_, err := sut.ShortenURL(ctx, request)

			assert.Error(t, err)
			assertCode(t, codes.Internal, err)
		})
	})
}

func setNewUser(ctx context.Context) context.Context {
	user := customctx.NewUser(domain.NewUserID(), true)
	return customctx.SetUser(ctx, user)
}

func assertCode(t *testing.T, want codes.Code, err error) {
	t.Helper()

	status, ok := status.FromError(err)
	require.True(t, ok)
	got := status.Code()
	assert.Equal(t, want, got)
}

func assertRedirectURL(t *testing.T, url string, urlStore domain.URLStore) {
	t.Helper()

	urlPath, err := getURLPath(url)
	require.NoError(t, err)

	got, err := urlStore.GetOriginalURL(context.Background(), strings.Trim(urlPath, "/"))
	require.NoError(t, err)

	assert.Equal(t, testURL, got)
}

func getURLPath(rawURL string) (string, error) {
	url, err := url.Parse(rawURL)

	if err != nil {
		return "", fmt.Errorf("get url path: %w", err)
	}

	return url.Path, nil
}
