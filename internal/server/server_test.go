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

	"github.com/nestjam/yap-shortener/internal/domain"
	"github.com/nestjam/yap-shortener/internal/persistance/inmemory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	testURL               = "https://practicum.yandex.ru/"
	baseURL               = "http://localhost:8080"
	acceptEncodingHeader  = "Accept-Encoding"
	contentEncodingHeader = "Content-Encoding"
	gzipEncoding          = "gzip"
	apiShortenPath        = "/api/shorten"
	apiBatchShortenPath   = "/api/shorten/batch"
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
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			err := urlStore.Add(context.Background(), shortURL, testURL)
			require.NoError(t, err)
			sut := New(urlStore, baseURL, zap.NewNop())
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
			sut := New(urlStore, baseURL, zap.NewNop())

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusMethodNotAllowed, response.Code)
			assertLocation(t, "", response)
			assertBody(t, "", response)
		})

		t.Run("url not found", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL, zap.NewNop())
			request := newGetRequest("EwHXdJfB")
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusNotFound, response.Code)
			assertLocation(t, "", response)
			assertBody(t, "not found", response)
		})
	})

	t.Run("shortening url", func(t *testing.T) {
		t.Run("shorten url", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL, zap.NewNop())
			request := newShortenRequest(testURL)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusCreated, response.Code)
			assertRedirectURL(t, response.Body.String(), urlStore)
		})

		t.Run("content type is not text-plain", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL, zap.NewNop())
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
			sut := New(urlStore, baseURL, zap.NewNop())
			request := newShortenRequest(testURL)
			response := httptest.NewRecorder()

			//1
			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusCreated, response.Code)
			assertRedirectURL(t, response.Body.String(), urlStore)

			request = newShortenRequest(testURL)
			response = httptest.NewRecorder()

			//2
			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusCreated, response.Code)
			assertRedirectURL(t, response.Body.String(), urlStore)
		})

		t.Run("url is empty", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL, zap.NewNop())
			request := newShortenRequest("")
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusBadRequest, response.Code)
			assertBody(t, urlIsEmptyMessage, response)
		})

		t.Run("client accepts br and gzip encodings", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL, zap.NewNop())
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
			sut := New(urlStore, baseURL, zap.NewNop())
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
			failingURLStore.AddFunc = func(ctx context.Context, shortURL, url string) error {
				return errors.New("failed to add url")
			}
			sut := New(failingURLStore, baseURL, zap.NewNop())
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
			sut := New(urlStore, baseURL, zap.NewNop())
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
			sut := New(urlStore, baseURL, zap.NewNop())
			request := newShortenAPIRequest(t, testURL)
			request.Header.Set(contentTypeHeader, textPlain)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusUnsupportedMediaType, response.Code)
		})

		t.Run("shorten same url twice", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL, zap.NewNop())
			request := newShortenAPIRequest(t, testURL)
			response := httptest.NewRecorder()

			//1
			sut.ServeHTTP(response, request)

			got := getShortURL(t, response.Body)
			assert.Equal(t, http.StatusCreated, response.Code)
			assertRedirectURL(t, got, urlStore)

			request = newShortenAPIRequest(t, testURL)
			response = httptest.NewRecorder()

			//2
			sut.ServeHTTP(response, request)

			got = getShortURL(t, response.Body)
			assert.Equal(t, http.StatusCreated, response.Code)
			assertRedirectURL(t, got, urlStore)
		})

		t.Run("url is empty", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL, zap.NewNop())
			request := newShortenAPIRequest(t, "")
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusBadRequest, response.Code)
			assertBody(t, urlIsEmptyMessage, response)
		})

		t.Run("request json is invalid", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL, zap.NewNop())
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
			sut := New(urlStore, baseURL, zap.NewNop())
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
			sut := New(urlStore, baseURL, zap.NewNop())
			request := newShortenAPIRequest(t, testURL)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assertContentEncoding(t, "", response)
		})

		t.Run("client sends encoded content", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL, zap.NewNop())
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
			failingURLStore.AddFunc = func(ctx context.Context, shortURL, url string) error {
				return errors.New("failed to add url")
			}
			sut := New(failingURLStore, baseURL, zap.NewNop())
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
		sut := New(urlStore, baseURL, zap.NewNop())
		request := newPutRequest(testURL)
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assert.Equal(t, want, response.Code)
	})

	t.Run("pinging service", func(t *testing.T) {
		t.Run("service is avaiable", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL, zap.NewNop())
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
			sut := New(unavailableURLStore, baseURL, zap.NewNop())
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
			sut := New(urlStore, baseURL, zap.NewNop())
			originalURLs := newBatch([]string{"https://practicum.yandex.ru/", "https://google.com/"})
			request := newBatchShortenAPIRequest(t, originalURLs)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusCreated, response.Code)
			assertContentType(t, applicationJSON, response)
			assertShortURLs(t, originalURLs, response.Body, urlStore)
		})

		t.Run("content type is not application-json", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL, zap.NewNop())
			originalURLs := newBatch([]string{testURL})
			request := newBatchShortenAPIRequest(t, originalURLs)
			request.Header.Set(contentTypeHeader, textPlain)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusUnsupportedMediaType, response.Code)
		})

		t.Run("shorten same url twice", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL, zap.NewNop())
			originalURLs := newBatch([]string{testURL})

			//1
			request := newBatchShortenAPIRequest(t, originalURLs)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusCreated, response.Code)
			assertShortURLs(t, originalURLs, response.Body, urlStore)

			//2
			request = newBatchShortenAPIRequest(t, originalURLs)
			response = httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusCreated, response.Code)
			assertShortURLs(t, originalURLs, response.Body, urlStore)
		})

		t.Run("batch is empty", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL, zap.NewNop())
			originalURLs := newBatch([]string{})
			request := newBatchShortenAPIRequest(t, originalURLs)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusBadRequest, response.Code)
			assertBody(t, batchIsEmptyMessage, response)
		})

		t.Run("request json is invalid", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL, zap.NewNop())
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
			sut := New(urlStore, baseURL, zap.NewNop())
			originalURLs := newBatch([]string{testURL})
			request := newBatchShortenAPIRequest(t, originalURLs)
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
			sut := New(urlStore, baseURL, zap.NewNop())
			originalURLs := newBatch([]string{testURL})
			request := newBatchShortenAPIRequest(t, originalURLs)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assertContentEncoding(t, "", response)
		})

		t.Run("client sends encoded content", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			sut := New(urlStore, baseURL, zap.NewNop())
			originalURLs := newBatch([]string{testURL})
			request := newBatchShortenAPIRequest(t, originalURLs)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusCreated, response.Code)
			assertShortURLs(t, originalURLs, response.Body, urlStore)
		})

		t.Run("failed to store url", func(t *testing.T) {
			urlStore, cleanup := u.CreateDependencies()
			t.Cleanup(cleanup)
			failingURLStore := domain.NewURLStoreDelegate(urlStore)
			failingURLStore.AddBatchFunc = func(ctx context.Context, pairs []domain.URLPair) error {
				return errors.New("failed to add url")
			}
			sut := New(failingURLStore, baseURL, zap.NewNop())
			originalURLs := newBatch([]string{testURL})
			request := newBatchShortenAPIRequest(t, originalURLs)
			response := httptest.NewRecorder()

			sut.ServeHTTP(response, request)

			assert.Equal(t, http.StatusInternalServerError, response.Code)
		})
	})
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

		got, err := store.Get(context.Background(), strings.Trim(urlPath, "/"))

		require.NoError(t, err)

		assert.Equal(t, got, req[i].URL)
	}
}

func newBatch(urls []string) []OriginalURL {
	batch := make([]OriginalURL, len(urls))
	for i := 0; i < len(urls); i++ {
		batch[i] = OriginalURL{URL: urls[i], CorrelationID: strconv.Itoa(i)}
	}
	return batch
}

func newBatchShortenAPIRequest(t *testing.T, r []OriginalURL) *http.Request {
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

	got, err := urlStore.Get(context.Background(), strings.Trim(urlPath, "/"))

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
