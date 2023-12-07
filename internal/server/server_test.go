package server

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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
	pingPath              = "ping"
)

type want struct {
	location string
	code     int
	body     string
}

type testURLStore struct {
	m            map[string]string
	baseURL      string
	lastShortURL string
	isAvailable  bool
}

func NewTestStore(baseURL string) *testURLStore {
	return &testURLStore{
		m:           map[string]string{},
		baseURL:     baseURL,
		isAvailable: true,
	}
}

func (t *testURLStore) Get(shortURL string) (string, error) {
	v, ok := t.m[shortURL]
	if !ok {
		return "", domain.ErrNotFound
	}
	return v, nil
}

func (t *testURLStore) Add(shortURL, url string) {
	t.m[shortURL] = url
	t.lastShortURL = t.baseURL + "/" + shortURL
}

func (t *testURLStore) IsAvailable() bool {
	return t.isAvailable
}

func TestRedirect(t *testing.T) {
	t.Run("redirect to stored url", func(t *testing.T) {
		want := want{
			code:     http.StatusTemporaryRedirect,
			location: testURL,
			body:     "<a href=\"https://practicum.yandex.ru/\">Temporary Redirect</a>.",
		}
		testStore := NewTestStore(baseURL)
		testStore.m["EwHXdJfB"] = testURL
		sut := New(testStore, baseURL, zap.NewNop())
		request := newGetRequest("EwHXdJfB")
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assert.Equal(t, want.code, response.Code)
		assertLocation(t, want.location, response)
		assertErrorMessage(t, want.body, response)
	})

	t.Run("path is empty", func(t *testing.T) {
		want := want{
			code:     http.StatusMethodNotAllowed,
			location: "",
		}
		request := newGetRequest("")
		response := httptest.NewRecorder()
		testStore := NewTestStore(baseURL)
		sut := New(testStore, baseURL, zap.NewNop())

		sut.ServeHTTP(response, request)

		assert.Equal(t, want.code, response.Code)
		assertLocation(t, want.location, response)
		assertBody(t, want.body, response)
	})

	t.Run("url not found", func(t *testing.T) {
		want := want{
			code: http.StatusNotFound,
			body: "not found",
		}
		testStore := NewTestStore(baseURL)
		sut := New(testStore, baseURL, zap.NewNop())
		request := newGetRequest("EwHXdJfB")
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assert.Equal(t, want.code, response.Code)
		assertLocation(t, want.location, response)
		assertErrorMessage(t, want.body, response)
	})

	t.Run("url not found (in-memory store)", func(t *testing.T) {
		want := want{
			code: http.StatusNotFound,
			body: "not found",
		}
		testStore := inmemory.New()
		sut := New(testStore, baseURL, zap.NewNop())
		request := newGetRequest("EwHXdJfB")
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assert.Equal(t, want.code, response.Code)
		assertLocation(t, want.location, response)
		assertErrorMessage(t, want.body, response)
	})
}

func TestShorten(t *testing.T) {
	t.Run("shorten url", func(t *testing.T) {
		want := want{
			code: http.StatusCreated,
		}
		testStore := NewTestStore(baseURL)
		sut := New(testStore, baseURL, zap.NewNop())
		request := newShortenRequest(testURL)
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assert.Equal(t, want.code, response.Code)
		assertBody(t, testStore.lastShortURL, response)
	})

	t.Run("content type is not text/plain", func(t *testing.T) {
		want := want{
			code: http.StatusUnsupportedMediaType,
		}
		testStore := NewTestStore(baseURL)
		sut := New(testStore, baseURL, zap.NewNop())
		request := newShortenRequest(testURL)
		request.Header.Set(contentTypeHeader, "application/xml")
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assert.Equal(t, want.code, response.Code)
		assertBody(t, want.body, response)
	})

	t.Run("shorten same url twice", func(t *testing.T) {
		want := want{
			code: http.StatusCreated,
		}
		testStore := NewTestStore(baseURL)
		sut := New(testStore, baseURL, zap.NewNop())
		request := newShortenRequest(testURL)
		response := httptest.NewRecorder()

		//1
		sut.ServeHTTP(response, request)

		assert.Equal(t, want.code, response.Code)
		assertBody(t, testStore.lastShortURL, response)

		request = newShortenRequest(testURL)
		response = httptest.NewRecorder()

		//2
		sut.ServeHTTP(response, request)

		assert.Equal(t, want.code, response.Code)
		assertBody(t, testStore.lastShortURL, response)
	})

	t.Run("url is empty", func(t *testing.T) {
		want := want{
			code: http.StatusBadRequest,
			body: urlIsEmptyMessage,
		}
		testStore := NewTestStore(baseURL)
		sut := New(testStore, baseURL, zap.NewNop())
		request := newShortenRequest("")
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assert.Equal(t, want.code, response.Code)
		assertErrorMessage(t, want.body, response)
	})

	t.Run("client accepts br and gzip encodings", func(t *testing.T) {
		testStore := NewTestStore(baseURL)
		sut := New(testStore, baseURL, zap.NewNop())
		request := newShortenRequest(testURL)
		request.Header.Set(acceptEncodingHeader, "br, "+gzipEncoding)
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assert.Equal(t, http.StatusCreated, response.Code)
		assertContentEncoding(t, gzipEncoding, response)
		got := getDecoded(t, response.Body)
		assert.Equal(t, testStore.lastShortURL, got)
	})

	t.Run("client sends content type x-gzip", func(t *testing.T) {
		testStore := NewTestStore(baseURL)
		sut := New(testStore, baseURL, zap.NewNop())
		request := newEncodedShortenRequest(t, testURL)
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assert.Equal(t, http.StatusCreated, response.Code)
		assertContentEncoding(t, "", response)
		assertBody(t, testStore.lastShortURL, response)
	})
}

func TestShortenAPI(t *testing.T) {
	t.Run("shorten url", func(t *testing.T) {
		testStore := NewTestStore(baseURL)
		sut := New(testStore, baseURL, zap.NewNop())
		request := newShortenAPIRequest(t, testURL)
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		body := response.Body.String()
		got := getShortURL(t, response.Body)
		assert.Equal(t, http.StatusCreated, response.Code)
		assert.Equal(t, testStore.lastShortURL, got)
		assertContentType(t, applicationJSON, response)
		assertContentLenght(t, len(body), response)
	})

	t.Run("content type is not application/json", func(t *testing.T) {
		testStore := NewTestStore(baseURL)
		sut := New(testStore, baseURL, zap.NewNop())
		request := newShortenAPIRequest(t, testURL)
		request.Header.Set(contentTypeHeader, textPlain)
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assert.Equal(t, http.StatusUnsupportedMediaType, response.Code)
	})

	t.Run("shorten same url twice", func(t *testing.T) {
		testStore := NewTestStore(baseURL)
		sut := New(testStore, baseURL, zap.NewNop())
		request := newShortenAPIRequest(t, testURL)
		response := httptest.NewRecorder()

		//1
		sut.ServeHTTP(response, request)

		got := getShortURL(t, response.Body)
		assert.Equal(t, http.StatusCreated, response.Code)
		assert.Equal(t, testStore.lastShortURL, got)

		request = newShortenAPIRequest(t, testURL)
		response = httptest.NewRecorder()

		//2
		sut.ServeHTTP(response, request)

		got = getShortURL(t, response.Body)
		assert.Equal(t, http.StatusCreated, response.Code)
		assert.Equal(t, testStore.lastShortURL, got)
	})

	t.Run("url is empty", func(t *testing.T) {
		testStore := NewTestStore(baseURL)
		sut := New(testStore, baseURL, zap.NewNop())
		request := newShortenAPIRequest(t, "")
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assert.Equal(t, http.StatusBadRequest, response.Code)
		assertErrorMessage(t, urlIsEmptyMessage, response)
	})

	t.Run("request json is invalid", func(t *testing.T) {
		testStore := NewTestStore(baseURL)
		sut := New(testStore, baseURL, zap.NewNop())
		request := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader("{{]}"))
		request.Header.Set(contentTypeHeader, applicationJSON)
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assert.Equal(t, http.StatusBadRequest, response.Code)
		assertErrorMessage(t, "failed to parse request", response)
	})

	t.Run("client accepts br and gzip encodings", func(t *testing.T) {
		testStore := NewTestStore(baseURL)
		sut := New(testStore, baseURL, zap.NewNop())
		request := newShortenAPIRequest(t, testURL)
		request.Header.Set(acceptEncodingHeader, "br, "+gzipEncoding)
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assert.Equal(t, http.StatusCreated, response.Code)
		assertContentEncoding(t, gzipEncoding, response)
		got := getShortURL(t, decodeResponse(t, response))
		assert.Equal(t, testStore.lastShortURL, got)
	})

	t.Run("client does not accept encoding", func(t *testing.T) {
		testStore := NewTestStore(baseURL)
		sut := New(testStore, baseURL, zap.NewNop())
		request := newShortenAPIRequest(t, testURL)
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assertContentEncoding(t, "", response)
	})

	t.Run("client sends encoded content", func(t *testing.T) {
		testStore := NewTestStore(baseURL)
		sut := New(testStore, baseURL, zap.NewNop())
		request := newEncodedShortenAPIRequest(t, testURL)
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assert.Equal(t, http.StatusCreated, response.Code)
		got := getShortURL(t, response.Body)
		assert.Equal(t, testStore.lastShortURL, got)
	})
}

func TestServeHTTP(t *testing.T) {
	t.Run("put method not allowed", func(t *testing.T) {
		want := http.StatusMethodNotAllowed
		testStore := NewTestStore(baseURL)
		sut := New(testStore, baseURL, zap.NewNop())
		request := newPutRequest(testURL)
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assert.Equal(t, want, response.Code)
	})
}

func TestPing(t *testing.T) {
	t.Run("service is avaiable", func(t *testing.T) {
		testStore := NewTestStore(baseURL)
		sut := New(testStore, baseURL, zap.NewNop())
		request := newPingRequest()
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assert.Equal(t, http.StatusOK, response.Code)
	})

	t.Run("service is not avaiable", func(t *testing.T) {
		testStore := NewTestStore(baseURL)
		testStore.isAvailable = false
		sut := New(testStore, baseURL, zap.NewNop())
		request := newPingRequest()
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assert.Equal(t, http.StatusInternalServerError, response.Code)
	})
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
	assert.Equal(t, want, r.Body.String())
}

func assertErrorMessage(t *testing.T, want string, r *httptest.ResponseRecorder) {
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
