package server

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nestjam/yap-shortener/internal/auth"
	"github.com/nestjam/yap-shortener/internal/domain"
	"github.com/nestjam/yap-shortener/internal/persistance/inmemory"
)

const (
	testURL               = "https://practicum.yandex.ru/"
	baseURL               = "http://localhost:8080"
	acceptEncodingHeader  = "Accept-Encoding"
	contentEncodingHeader = "Content-Encoding"
	gzipEncoding          = "gzip"
	apiShortenPath        = "/api/shorten"
	apiBatchShortenPath   = "/api/shorten/batch"
	userURLsPath          = "/api/user/urls"
	pingPath              = "ping"
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
	t.Run("getting original url", func(t *testing.T) {
		t.Run("redirect to original url", func(t *testing.T) {
			const (
				shortURL = "EwHXdJfB"
				body     = "<a href=\"https://practicum.yandex.ru/\">Temporary Redirect</a>."
			)
			userID := domain.NewUserID()
			pair := domain.URLPair{
				ShortURL:    shortURL,
				OriginalURL: testURL,
			}
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			err := urlStore.AddURL(context.Background(), pair, userID)
			require.NoError(t, err)
			sut := New(urlStore, baseURL)
			request := newGetRequest(shortURL)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusTemporaryRedirect, response.Code)
			assertLocation(t, testURL, response)
			assertBody(t, body, response)
		})

		t.Run("path is empty", func(t *testing.T) {
			request := newGetRequest("")
			response := httptest.NewRecorder()
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusMethodNotAllowed, response.Code)
			assertLocation(t, "", response)
			assertBody(t, "", response)
		})

		t.Run("url not found", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			request := newGetRequest("EwHXdJfB")
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusNotFound, response.Code)
			assertLocation(t, "", response)
			assertBody(t, "not found", response)
		})

		t.Run("url is deleted", func(t *testing.T) {
			const (
				shortURL = "EwHXdJfB"
				body     = "<a href=\"https://practicum.yandex.ru/\">Temporary Redirect</a>."
			)
			ctx := context.Background()
			userID := domain.NewUserID()
			pair := domain.URLPair{
				ShortURL:    shortURL,
				OriginalURL: testURL,
			}
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			err := urlStore.AddURL(ctx, pair, userID)
			require.NoError(t, err)

			err = urlStore.DeleteUserURLs(ctx, []string{pair.ShortURL}, userID)
			require.NoError(t, err)

			sut := New(urlStore, baseURL)
			request := newGetRequest(shortURL)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusGone, response.Code)
			assertLocation(t, "", response)
			assertBody(t, "url is deleted", response)
		})
	})

	t.Run("shortening url", func(t *testing.T) {
		t.Run("shorten url", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			request := newShortenRequest(testURL)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusCreated, response.Code)
			assertRedirectURL(t, response.Body.String(), urlStore)
		})

		t.Run("content type is not text-plain", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			request := newShortenRequest(testURL)
			request.Header.Set(contentTypeHeader, "application/xml")
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusUnsupportedMediaType, response.Code)
			assertBody(t, "", response)
		})

		t.Run("shorten same url twice", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)

			//1
			request := newShortenRequest(testURL)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusCreated, response.Code)
			assertRedirectURL(t, response.Body.String(), urlStore)

			//2
			request = newShortenRequest(testURL)
			response = httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusConflict, response.Code)
			assertRedirectURL(t, response.Body.String(), urlStore)
		})

		t.Run("url is empty", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			request := newShortenRequest("")
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusBadRequest, response.Code)
			assertBody(t, urlIsEmptyMessage, response)
		})

		t.Run("client accepts br and gzip encodings", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			request := newShortenRequest(testURL)
			request.Header.Set(acceptEncodingHeader, "br, "+gzipEncoding)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusCreated, response.Code)
			assertContentEncoding(t, gzipEncoding, response)
			got := getDecoded(t, response.Body)
			assertRedirectURL(t, got, urlStore)
		})

		t.Run("client sends content type x-gzip", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			request := newEncodedShortenRequest(t, testURL)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusCreated, response.Code)
			assertContentEncoding(t, "", response)
			assertRedirectURL(t, response.Body.String(), urlStore)
		})

		t.Run("failed to store url", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			failingURLStore := domain.NewURLStoreDelegate(urlStore)
			failingURLStore.AddURLFunc = func(ctx context.Context, pair domain.URLPair, userID domain.UserID) error {
				return errors.New("failed to add url")
			}
			sut := New(failingURLStore, baseURL)
			request := newShortenRequest(testURL)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusInternalServerError, response.Code)
		})
	})

	t.Run("shortening url (api)", func(t *testing.T) {
		t.Run("shorten url", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			request := newShortenAPIRequest(t, testURL)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			body := response.Body.String()
			got := getShortURL(t, response.Body)
			assert.Equal(t, http.StatusCreated, response.Code)
			assertRedirectURL(t, got, urlStore)
			assertContentType(t, applicationJSON, response)
			assertContentLenght(t, len(body), response)
		})

		t.Run("content type is not application-json", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			request := newShortenAPIRequest(t, testURL)
			request.Header.Set(contentTypeHeader, textPlain)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusUnsupportedMediaType, response.Code)
		})

		t.Run("shorten same url twice", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)

			//1
			request := newShortenAPIRequest(t, testURL)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			got := getShortURL(t, response.Body)
			assert.Equal(t, http.StatusCreated, response.Code)
			assertRedirectURL(t, got, urlStore)

			//2
			request = newShortenAPIRequest(t, testURL)
			response = httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusConflict, response.Code)
			got = getShortURL(t, response.Body)
			assertRedirectURL(t, got, urlStore)
		})

		t.Run("url is empty", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			request := newShortenAPIRequest(t, "")
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusBadRequest, response.Code)
			assertBody(t, urlIsEmptyMessage, response)
		})

		t.Run("request json is invalid", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			request := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader("{{]}"))
			request.Header.Set(contentTypeHeader, applicationJSON)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusBadRequest, response.Code)
			assertBody(t, "failed to parse request", response)
		})

		t.Run("client accepts br and gzip encodings", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			request := newShortenAPIRequest(t, testURL)
			request.Header.Set(acceptEncodingHeader, "br, "+gzipEncoding)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusCreated, response.Code)
			assertContentEncoding(t, gzipEncoding, response)
			got := getShortURL(t, decodeResponse(t, response))
			assertRedirectURL(t, got, urlStore)
		})

		t.Run("client does not accept encoding", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			request := newShortenAPIRequest(t, testURL)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assertContentEncoding(t, "", response)
		})

		t.Run("client sends encoded content", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			request := newEncodedShortenAPIRequest(t, testURL)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusCreated, response.Code)
			got := getShortURL(t, response.Body)
			assertRedirectURL(t, got, urlStore)
		})

		t.Run("failed to store url", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			failingURLStore := domain.NewURLStoreDelegate(urlStore)
			failingURLStore.AddURLFunc = func(ctx context.Context, pair domain.URLPair, userID domain.UserID) error {
				return errors.New("failed to add url")
			}
			sut := New(failingURLStore, baseURL)
			request := newEncodedShortenAPIRequest(t, testURL)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusInternalServerError, response.Code)
		})
	})

	t.Run("put method not allowed", func(t *testing.T) {
		want := http.StatusMethodNotAllowed
		urlStore, cleanup := u.CreateDependencies()
		t.Cleanup(cleanup)
		sut := New(urlStore, baseURL)
		request := newPutRequest(testURL)
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assert.Equal(t, want, response.Code)
	})

	t.Run("pinging service", func(t *testing.T) {
		t.Run("service is avaiable", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			request := newPingRequest()
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusOK, response.Code)
		})

		t.Run("service is not avaiable", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			unavailableURLStore := domain.NewURLStoreDelegate(urlStore)
			unavailableURLStore.IsAvailableFunc = func(ctx context.Context) bool {
				return false
			}
			sut := New(unavailableURLStore, baseURL)
			request := newPingRequest()
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusInternalServerError, response.Code)
		})
	})

	t.Run("batch shortening urls (api)", func(t *testing.T) {
		t.Run("shorten urls", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			originalURLs := newBatch([]string{"https://practicum.yandex.ru/", "https://google.com/"})
			request := newShortenURLsAPIRequest(t, originalURLs)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusCreated, response.Code)
			assertContentType(t, applicationJSON, response)
			assertShortURLs(t, originalURLs, response.Body, urlStore)
		})

		t.Run("content type is not application-json", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			originalURLs := newBatch([]string{testURL})
			request := newShortenURLsAPIRequest(t, originalURLs)
			request.Header.Set(contentTypeHeader, textPlain)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusUnsupportedMediaType, response.Code)
		})

		t.Run("shorten same url twice", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			originalURLs := newBatch([]string{testURL})

			//1
			request := newShortenURLsAPIRequest(t, originalURLs)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusCreated, response.Code)
			assertShortURLs(t, originalURLs, response.Body, urlStore)

			//2
			request = newShortenURLsAPIRequest(t, originalURLs)
			response = httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusCreated, response.Code)
			assertShortURLs(t, originalURLs, response.Body, urlStore)
		})

		t.Run("batch is empty", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			originalURLs := newBatch([]string{})
			request := newShortenURLsAPIRequest(t, originalURLs)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusBadRequest, response.Code)
			assertBody(t, batchIsEmptyMessage, response)
		})

		t.Run("url is empty", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			originalURLs := newBatch([]string{""})
			request := newShortenURLsAPIRequest(t, originalURLs)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusBadRequest, response.Code)
			assertBody(t, urlIsEmptyMessage, response)
		})

		t.Run("request json is invalid", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			request := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader("{{]}"))
			request.Header.Set(contentTypeHeader, applicationJSON)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusBadRequest, response.Code)
			assertBody(t, "failed to parse request", response)
		})

		t.Run("client accepts br and gzip encodings", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			originalURLs := newBatch([]string{testURL})
			request := newShortenURLsAPIRequest(t, originalURLs)
			request.Header.Set(acceptEncodingHeader, "br, "+gzipEncoding)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusCreated, response.Code)
			assertContentEncoding(t, gzipEncoding, response)
			assertShortURLs(t, originalURLs, decodeResponse(t, response), urlStore)
		})

		t.Run("client does not accept encoding", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			originalURLs := newBatch([]string{testURL})
			request := newShortenURLsAPIRequest(t, originalURLs)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assertContentEncoding(t, "", response)
		})

		t.Run("client sends encoded content", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			originalURLs := newBatch([]string{testURL})
			request := newShortenURLsAPIRequest(t, originalURLs)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusCreated, response.Code)
			assertShortURLs(t, originalURLs, response.Body, urlStore)
		})

		t.Run("failed to store url", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			failingURLStore := domain.NewURLStoreDelegate(urlStore)
			failingURLStore.AddURLsFunc = func(ctx context.Context, pairs []domain.URLPair, userID domain.UserID) error {
				return errors.New("failed to add url")
			}
			sut := New(failingURLStore, baseURL)
			originalURLs := newBatch([]string{testURL})
			request := newShortenURLsAPIRequest(t, originalURLs)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusInternalServerError, response.Code)
		})

		t.Run("request posts for too many urls", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL, WithShortenURLsMaxCount(2))
			urls := []string{
				"foo.net",
				"bar.net",
				"buz.net",
			}
			originalURLs := newBatch(urls)

			request := newShortenURLsAPIRequest(t, originalURLs)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusForbidden, response.Code)
			assertBody(t, "too many urls", response)
		})
	})

	t.Run("get user urls", func(t *testing.T) {
		t.Run("get urls shortened by user", func(t *testing.T) {
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
			err := urlStore.AddURLs(context.Background(), userURLs, userID)
			require.NoError(t, err)
			sut := New(urlStore, baseURL)
			request := newGetUserURLsRequest(t, userID)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusOK, response.Code)
			assertContentType(t, applicationJSON, response)
			assertUserURLs(t, userURLs, response.Body)
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
			err := urlStore.AddURLs(context.Background(), userURLs, otherUserID)
			require.NoError(t, err)
			sut := New(urlStore, baseURL)
			userID := domain.NewUserID()
			request := newGetUserURLsRequest(t, userID)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusNoContent, response.Code)
			assertBody(t, "no urls", response)
		})

		t.Run("user is not authorized", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			request := httptest.NewRequest(http.MethodGet, userURLsPath, nil)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusUnauthorized, response.Code)
		})
	})

	t.Run("delete user urls", func(t *testing.T) {
		t.Run("delete urls shortened by user", func(t *testing.T) {
			ctx := context.Background()
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
			err := urlStore.AddURLs(ctx, userURLs, userID)
			require.NoError(t, err)
			sut := New(urlStore, baseURL)
			request := newDeleteUserURLsRequest(t, userURLs, userID)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusAccepted, response.Code)
			pairs, err := urlStore.GetUserURLs(ctx, userID)
			require.NoError(t, err)
			assert.Equal(t, 0, len(pairs))
		})

		t.Run("request content is invalid", func(t *testing.T) {
			userID := domain.NewUserID()
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL)
			request := newDeleteUserURLsInvalidRequest(t, userID)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusBadRequest, response.Code)
		})

		t.Run("failed to delete user urls", func(t *testing.T) {
			userID := domain.NewUserID()
			userURLs := []domain.URLPair{}
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			failingURLStore := domain.NewURLStoreDelegate(urlStore)
			failingURLStore.DeleteUserURLsFunc = func(ctx context.Context, urls []string, userID domain.UserID) error {
				return errors.New("failed to delete urls")
			}
			sut := New(failingURLStore, baseURL)
			request := newDeleteUserURLsRequest(t, userURLs, userID)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusInternalServerError, response.Code)
			assertBody(t, "failed to delete user urls", response)
		})
	})

	t.Run("get internal stats", func(t *testing.T) {
		t.Run("get internal stats from trusted subnet", func(t *testing.T) {
			ctx := context.Background()
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
			want := Stats{
				URLs:  len(userURLs),
				Users: 1,
			}
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			err := urlStore.AddURLs(ctx, userURLs, userID)
			require.NoError(t, err)
			sut := New(urlStore, baseURL, WithTrustedSubnet("127.0.0.0/24"))
			const ip = "127.0.0.1"
			request := newGetInternalStatRequest(t, ip)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			require.Equal(t, http.StatusOK, response.Code)
			assertStats(t, want, response.Body)
		})

		t.Run("trusted subnet is not set", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL, WithTrustedSubnet(""))
			const ip = "127.0.0.1"
			request := newGetInternalStatRequest(t, ip)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusForbidden, response.Code)
		})

		t.Run("client ip does not belong to trusted subnet", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL, WithTrustedSubnet("127.0.0.0/24"))
			const ip = "150.172.238.178"
			request := newGetInternalStatRequest(t, ip)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusForbidden, response.Code)
		})

		t.Run("trusted subnet address is invalid", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL, WithTrustedSubnet("12700.0.0.0/24"))
			const ip = "127.0.0.3"
			request := newGetInternalStatRequest(t, ip)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusInternalServerError, response.Code)
		})

		t.Run("x-real-ip is invalid", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL, WithTrustedSubnet("127.0.0.0/24"))
			const ip = "12700.0.0.3"
			request := newGetInternalStatRequest(t, ip)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusForbidden, response.Code)
		})

		t.Run("store failed to execute query", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			failingURLStore := domain.NewURLStoreDelegate(urlStore)
			failingURLStore.GetURLsAndUsersCountFunc = func(ctx context.Context) (urlsCount, usersCount int, err error) {
				err = errors.New("failed to execute query")
				return
			}
			sut := New(failingURLStore, baseURL, WithTrustedSubnet("127.0.0.0/24"))
			const ip = "127.0.0.4"
			request := newGetInternalStatRequest(t, ip)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusInternalServerError, response.Code)
		})
	})
}

func assertUserURLs(t *testing.T, want []domain.URLPair, r io.Reader) {
	t.Helper()
	var got []UserURL
	err := json.NewDecoder(r).Decode(&got)

	require.NoError(t, err, "unable to parse response from server: %v", err)

	urls := make([]UserURL, len(want))
	for i := 0; i < len(want); i++ {
		urls[i].OriginalURL = want[i].OriginalURL
		urls[i].ShortURL = baseURL + "/" + want[i].ShortURL
	}
	assert.ElementsMatch(t, urls, got)
}

func assertShortURLs(t *testing.T, req []OriginalURL, r io.Reader, store domain.URLStore) {
	t.Helper()
	var resp []ShortURL
	err := json.NewDecoder(r).Decode(&resp)

	require.NoError(t, err, "unable to parse response from server: %v", err)

	assert.Equal(t, len(req), len(resp))
	for i := 0; i < len(req); i++ {
		urlPath, err := getURLPath(resp[i].URL)

		require.NoError(t, err)

		got, err := store.GetOriginalURL(context.Background(), strings.Trim(urlPath, "/"))

		require.NoError(t, err)

		assert.Equal(t, got, req[i].URL)
	}
}

func assertStats(t *testing.T, want Stats, r io.Reader) {
	t.Helper()

	var got Stats
	err := json.NewDecoder(r).Decode(&got)
	require.NoError(t, err, "unable to parse response from server: %v", err)

	assert.Equal(t, want, got)
}

func newBatch(urls []string) []OriginalURL {
	batch := make([]OriginalURL, len(urls))
	for i := 0; i < len(urls); i++ {
		batch[i] = OriginalURL{URL: urls[i], CorrelationID: strconv.Itoa(i)}
	}
	return batch
}

func newShortenURLsAPIRequest(t *testing.T, r []OriginalURL) *http.Request {
	t.Helper()
	body, err := json.Marshal(&r)

	require.NoError(t, err, "unable to marshal %q, %v", r, err)

	request := httptest.NewRequest(http.MethodPost, apiBatchShortenPath, strings.NewReader(string(body)))
	request.Header.Set(contentTypeHeader, applicationJSON)
	return request
}

func newPingRequest() *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/"+pingPath, nil)
	return r
}

func newGetRequest(shortURL string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/"+shortURL, nil)
	r.Header.Set(contentTypeHeader, textPlain+"; charset=utf-8")
	return r
}

func newShortenRequest(url string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(url))
	r.Header.Set(contentTypeHeader, textPlain+"; charset=utf-8")
	return r
}

func newEncodedShortenRequest(t *testing.T, url string) *http.Request {
	t.Helper()

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	defer func() {
		_ = gw.Close()
	}()
	_, err := gw.Write([]byte(url))

	require.NoError(t, err, "failed to write encoded: %v", err)

	r := httptest.NewRequest(http.MethodPost, "/", &buf)
	r.Header.Set(contentTypeHeader, applicationGZIP)
	r.Header.Set(contentEncodingHeader, gzipEncoding)
	return r
}

func newShortenAPIRequest(t *testing.T, url string) *http.Request {
	t.Helper()
	r := ShortenRequest{URL: url}
	body, err := json.Marshal(&r)

	require.NoError(t, err, "unable to marshal %q, %v", r, err)

	request := httptest.NewRequest(http.MethodPost, apiShortenPath, strings.NewReader(string(body)))
	request.Header.Set(contentTypeHeader, applicationJSON)
	return request
}

func newEncodedShortenAPIRequest(t *testing.T, url string) *http.Request {
	t.Helper()
	r := ShortenRequest{URL: url}
	body, err := json.Marshal(&r)

	require.NoError(t, err, "unable to marshal %q, %v", r, err)

	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	defer func() {
		_ = gw.Close()
	}()
	_, err = gw.Write(body)

	require.NoError(t, err, "failed to write encoded: %v", err)

	request := httptest.NewRequest(http.MethodPost, apiShortenPath, &buf)
	request.Header.Set(contentTypeHeader, applicationJSON)
	request.Header.Set(contentEncodingHeader, gzipEncoding)
	return request
}

func newPutRequest(url string) *http.Request {
	r := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(url))
	r.Header.Set(contentTypeHeader, textPlain+"; charset=utf-8")
	return r
}

func newGetUserURLsRequest(t *testing.T, userID domain.UserID) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, userURLsPath, nil)
	a := auth.New(secretKey, tokenExp)

	cookie, err := a.CreateCookie(userID)
	require.NoError(t, err)

	r.AddCookie(cookie)
	return r
}

func newDeleteUserURLsRequest(t *testing.T, userURLs []domain.URLPair, userID domain.UserID) *http.Request {
	t.Helper()

	shortURLs := make([]string, len(userURLs))
	for i := 0; i < len(userURLs); i++ {
		shortURLs[i] = userURLs[i].ShortURL
	}
	body, err := json.Marshal(&shortURLs)
	require.NoError(t, err)

	buf := bytes.NewBuffer(body)
	r := httptest.NewRequest(http.MethodDelete, userURLsPath, buf)
	r.Header.Set(contentTypeHeader, applicationJSON)
	a := auth.New(secretKey, tokenExp)

	cookie, err := a.CreateCookie(userID)
	require.NoError(t, err)

	r.AddCookie(cookie)
	return r
}

func newDeleteUserURLsInvalidRequest(t *testing.T, userID domain.UserID) *http.Request {
	t.Helper()

	buf := bytes.NewBufferString("[{ Invalid: true ]}")
	r := httptest.NewRequest(http.MethodDelete, userURLsPath, buf)
	r.Header.Set(contentTypeHeader, applicationJSON)
	a := auth.New(secretKey, tokenExp)

	cookie, err := a.CreateCookie(userID)
	require.NoError(t, err)

	r.AddCookie(cookie)
	return r
}

func newGetInternalStatRequest(t *testing.T, ip string) *http.Request {
	t.Helper()

	r := httptest.NewRequest(http.MethodGet, "/api/internal/stats", nil)
	r.Header.Set("X-Real-IP", ip)
	return r
}

func assertLocation(t *testing.T, want string, r *httptest.ResponseRecorder) {
	t.Helper()
	assert.Equal(t, want, r.Header().Get(locationHeader))
}

func assertBody(t *testing.T, want string, r *httptest.ResponseRecorder) {
	t.Helper()
	assert.Equal(t, want, strings.TrimSpace(r.Body.String()))
}

func assertContentType(t *testing.T, want string, r *httptest.ResponseRecorder) {
	t.Helper()
	assert.Equal(t, want, r.Header().Get(contentTypeHeader))
}

func assertContentLenght(t *testing.T, want int, r *httptest.ResponseRecorder) {
	t.Helper()
	assert.Equal(t, strconv.Itoa(want), r.Header().Get(contentLengthHeader))
}

func getShortURL(t *testing.T, r io.Reader) string {
	t.Helper()
	var resp ShortenResponse
	err := json.NewDecoder(r).Decode(&resp)

	require.NoError(t, err, "unable to parse response from server: %v", err)

	return resp.Result
}

func assertContentEncoding(t *testing.T, want string, r *httptest.ResponseRecorder) {
	t.Helper()
	assert.Equal(t, want, r.Header().Get(contentEncodingHeader))
}

func getDecoded(t *testing.T, r io.Reader) string {
	t.Helper()
	gz, err := gzip.NewReader(r)

	require.NoError(t, err, "failed to decode: %v", err)

	defer func() {
		_ = gz.Close()
	}()

	str, err := io.ReadAll(gz)

	require.NoError(t, err, "failed to read decoded: %v", err)

	return string(str)
}

func decodeResponse(t *testing.T, r *httptest.ResponseRecorder) *strings.Reader {
	t.Helper()
	return strings.NewReader(getDecoded(t, r.Body))
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
