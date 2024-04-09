package grpc

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nestjam/yap-shortener/internal/domain"
	"github.com/nestjam/yap-shortener/internal/persistance/inmemory"
	pb "github.com/nestjam/yap-shortener/proto"

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

	t.Run("batch shortening urls", func(t *testing.T) {
		t.Run("shorten urls", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			ctx := context.Background()
			ctx = setNewUser(ctx)
			originalURLs := newBatch([]string{"https://practicum.yandex.ru/", "https://google.com/"})
			request := &pb.ShortenURLsRequest{
				URLs: originalURLs,
			}

			resp, err := sut.ShortenURLs(ctx, request)

			require.NoError(t, err)
			assertShortenedURLs(t, request, resp, urlStore)
		})

		t.Run("shorten same url twice", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			ctx := context.Background()
			ctx = setNewUser(ctx)
			originalURLs := newBatch([]string{"https://practicum.yandex.ru/", "https://google.com/"})
			request := &pb.ShortenURLsRequest{
				URLs: originalURLs,
			}

			//1
			resp, err := sut.ShortenURLs(ctx, request)

			require.NoError(t, err)
			assertShortenedURLs(t, request, resp, urlStore)

			//2
			resp, err = sut.ShortenURLs(ctx, request)

			require.NoError(t, err)
			assertShortenedURLs(t, request, resp, urlStore)
		})

		t.Run("batch is empty", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			ctx := context.Background()
			ctx = setNewUser(ctx)
			originalURLs := newBatch([]string{})
			request := &pb.ShortenURLsRequest{
				URLs: originalURLs,
			}

			_, err := sut.ShortenURLs(ctx, request)

			require.Error(t, err)
			assertCode(t, codes.InvalidArgument, err)
		})

		t.Run("url is empty", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			ctx := context.Background()
			ctx = setNewUser(ctx)
			originalURLs := newBatch([]string{""})
			request := &pb.ShortenURLsRequest{
				URLs: originalURLs,
			}

			_, err := sut.ShortenURLs(ctx, request)

			require.Error(t, err)
			assertCode(t, codes.InvalidArgument, err)
		})

		t.Run("failed to store url", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			failingURLStore := domain.NewURLStoreDelegate(urlStore)
			failingURLStore.AddURLsFunc = func(ctx context.Context, pairs []domain.URLPair, userID domain.UserID) error {
				return errors.New("failed to add url")
			}
			sut := New(failingURLStore, baseURL)
			ctx := context.Background()
			ctx = setNewUser(ctx)
			originalURLs := newBatch([]string{testURL})
			request := &pb.ShortenURLsRequest{
				URLs: originalURLs,
			}

			_, err := sut.ShortenURLs(ctx, request)

			require.Error(t, err)
			assertCode(t, codes.Internal, err)
		})

		t.Run("request posts for too many urls", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL, WithShortenURLsMaxCount(2))
			ctx := context.Background()
			ctx = setNewUser(ctx)
			urls := []string{
				"foo.net",
				"bar.net",
				"buz.net",
			}
			originalURLs := newBatch(urls)
			request := &pb.ShortenURLsRequest{
				URLs: originalURLs,
			}

			_, err := sut.ShortenURLs(ctx, request)

			require.Error(t, err)
			assertCode(t, codes.InvalidArgument, err)
		})
	})

	t.Run("get user urls", func(t *testing.T) {
		t.Run("get urls shortened by user", func(t *testing.T) {
			userID := domain.NewUserID()
			urls := []domain.URLPair{
				{
					OriginalURL: "http://yandex.ru",
					ShortURL:    "123",
				},
				{
					OriginalURL: "http://mail.ru",
					ShortURL:    "456",
				},
			}
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			ctx := context.Background()
			ctx = setUser(ctx, userID)
			err := urlStore.AddURLs(ctx, urls, userID)
			require.NoError(t, err)
			request := &pb.GetUserURLsRequest{}

			resp, err := sut.GetUserURLs(ctx, request)

			require.NoError(t, err)
			assertUserURLs(t, urls, resp.URLs)
		})

		t.Run("no urls shortened by user", func(t *testing.T) {
			otherUserID := domain.NewUserID()
			userURLs := []domain.URLPair{
				{
					OriginalURL: "http://yandex.ru",
					ShortURL:    "123",
				},
			}
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			ctx := context.Background()
			ctx = setUser(ctx, domain.NewUserID())
			err := urlStore.AddURLs(context.Background(), userURLs, otherUserID)
			require.NoError(t, err)
			request := &pb.GetUserURLsRequest{}

			resp, err := sut.GetUserURLs(ctx, request)

			require.NoError(t, err)
			assert.Equal(t, 0, len(resp.URLs))
		})

		t.Run("user is not authorized", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			ctx := context.Background()
			ctx = setNewUser(ctx)
			request := &pb.GetUserURLsRequest{}

			_, err := sut.GetUserURLs(ctx, request)

			require.Error(t, err)
			assertCode(t, codes.PermissionDenied, err)
		})

		t.Run("url store failed to get urls", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			failingURLStore := domain.NewURLStoreDelegate(urlStore)
			failingURLStore.GetUserURLsFunc = func(ctx context.Context, userID domain.UserID) ([]domain.URLPair, error) {
				return nil, errors.New("failed to get urls")
			}
			sut := New(failingURLStore, baseURL)
			ctx := context.Background()
			ctx = setUser(ctx, domain.NewUserID())
			request := &pb.GetUserURLsRequest{}

			_, err := sut.GetUserURLs(ctx, request)

			require.Error(t, err)
			assertCode(t, codes.Internal, err)
		})
	})

	t.Run("delete user urls", func(t *testing.T) {
		t.Run("delete urls shortened by user", func(t *testing.T) {
			userID := domain.NewUserID()
			userURLs := []domain.URLPair{
				{
					OriginalURL: "http://yandex.ru",
					ShortURL:    "123",
				},
				{
					OriginalURL: "http://mail.ru",
					ShortURL:    "456",
				},
			}
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			ctx := context.Background()
			ctx = setUser(ctx, userID)
			err := urlStore.AddURLs(ctx, userURLs, userID)
			require.NoError(t, err)
			request := &pb.DeleteUserURLsRequest{
				Keys: []string{
					userURLs[0].ShortURL,
					userURLs[1].ShortURL,
				},
			}

			_, err = sut.GetDeleteUserURLs(ctx, request)

			require.NoError(t, err)
			pairs, err := urlStore.GetUserURLs(ctx, userID)
			require.NoError(t, err)
			assert.Equal(t, 0, len(pairs))
		})

		t.Run("failed to delete user urls", func(t *testing.T) {
			userID := domain.NewUserID()
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			failingURLStore := domain.NewURLStoreDelegate(urlStore)
			failingURLStore.DeleteUserURLsFunc = func(ctx context.Context, urls []string, userID domain.UserID) error {
				return errors.New("failed to delete urls")
			}
			sut := New(failingURLStore, baseURL)
			ctx := context.Background()
			ctx = setUser(ctx, userID)
			request := &pb.DeleteUserURLsRequest{}

			_, err := sut.GetDeleteUserURLs(ctx, request)

			require.Error(t, err)
			assertCode(t, codes.Internal, err)
		})
	})
}

func newBatch(urls []string) []*pb.CorrelatedURL {
	batch := make([]*pb.CorrelatedURL, len(urls))
	for i := 0; i < len(urls); i++ {
		batch[i] = &pb.CorrelatedURL{URL: urls[i], CorrelationID: strconv.Itoa(i)}
	}
	return batch
}

func setNewUser(ctx context.Context) context.Context {
	user := customctx.NewUser(domain.NewUserID(), true)
	return customctx.SetUser(ctx, user)
}

func setUser(ctx context.Context, userID domain.UserID) context.Context {
	user := customctx.NewUser(userID, false)
	return customctx.SetUser(ctx, user)
}

func getURLPath(rawURL string) (string, error) {
	url, err := url.Parse(rawURL)

	if err != nil {
		return "", fmt.Errorf("get url path: %w", err)
	}

	return url.Path, nil
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

//nolint:lll // naturally long name
func assertShortenedURLs(t *testing.T, req *pb.ShortenURLsRequest, resp *pb.ShortenURLsResponse, store domain.URLStore) {
	t.Helper()

	assert.Equal(t, len(req.URLs), len(resp.URLs))
	for i := 0; i < len(req.URLs); i++ {
		urlPath, err := getURLPath(resp.URLs[i].URL)
		require.NoError(t, err)

		got, err := store.GetOriginalURL(context.Background(), strings.Trim(urlPath, "/"))
		require.NoError(t, err)

		assert.Equal(t, got, req.URLs[i].URL)
	}
}

func assertUserURLs(t *testing.T, want []domain.URLPair, got []*pb.UserURL) {
	t.Helper()

	urls := make([]*pb.UserURL, len(want))
	for i := 0; i < len(want); i++ {
		urls[i] = &pb.UserURL{
			OriginalURL:  want[i].OriginalURL,
			ShortenedURL: baseURL + "/" + want[i].ShortURL,
		}
	}
	assert.ElementsMatch(t, urls, got)
}
