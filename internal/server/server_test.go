package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/nestjam/yap-shortener/internal/model"
	"github.com/nestjam/yap-shortener/internal/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testURL = "https://practicum.yandex.ru/"
	baseURL = "http://localhost:8080"
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
}

func NewTestStore(baseURL string) *testURLStore {
	return &testURLStore{
		m:       map[string]string{},
		baseURL: baseURL,
	}
}

func (t *testURLStore) Get(shortURL string) (string, error) {
	v, ok := t.m[shortURL]
	if !ok {
		return "", model.ErrNotFound
	}
	return v, nil
}

func (t *testURLStore) Add(shortURL, url string) {
	t.m[shortURL] = url
	t.lastShortURL = t.baseURL + "/" + shortURL
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
		sut := New(testStore, baseURL)
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
		sut := New(testStore, baseURL)

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
		sut := New(testStore, baseURL)
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
		testStore := store.NewInMemory()
		sut := New(testStore, baseURL)
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
		sut := New(testStore, baseURL)
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
		sut := New(testStore, baseURL)
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
		sut := New(testStore, baseURL)
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
		sut := New(testStore, baseURL)
		request := newShortenRequest("")
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assert.Equal(t, want.code, response.Code)
		assertErrorMessage(t, want.body, response)
	})
}

func TestAPIShorten(t *testing.T) {
	t.Run("shorten url", func(t *testing.T) {
		testStore := NewTestStore(baseURL)
		sut := New(testStore, baseURL)
		request := newAPIShortenRequest(t, testURL)
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		body := response.Body.String()
		got := getShortURLFromResponse(t, response)
		assert.Equal(t, http.StatusCreated, response.Code)
		assert.Equal(t, testStore.lastShortURL, got)
		assertContentType(t, applicationJSON, response)
		assertContentLenght(t, len(body), response)
	})

	t.Run("content type is not application/json", func(t *testing.T) {
		testStore := NewTestStore(baseURL)
		sut := New(testStore, baseURL)
		request := newAPIShortenRequest(t, testURL)
		request.Header.Set(contentTypeHeader, textPlain)
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assert.Equal(t, http.StatusUnsupportedMediaType, response.Code)
	})

	t.Run("shorten same url twice", func(t *testing.T) {
		testStore := NewTestStore(baseURL)
		sut := New(testStore, baseURL)
		request := newAPIShortenRequest(t, testURL)
		response := httptest.NewRecorder()

		//1
		sut.ServeHTTP(response, request)

		got := getShortURLFromResponse(t, response)
		assert.Equal(t, http.StatusCreated, response.Code)
		assert.Equal(t, testStore.lastShortURL, got)

		request = newAPIShortenRequest(t, testURL)
		response = httptest.NewRecorder()

		//2
		sut.ServeHTTP(response, request)

		got = getShortURLFromResponse(t, response)
		assert.Equal(t, http.StatusCreated, response.Code)
		assert.Equal(t, testStore.lastShortURL, got)
	})

	t.Run("url is empty", func(t *testing.T) {
		testStore := NewTestStore(baseURL)
		sut := New(testStore, baseURL)
		request := newAPIShortenRequest(t, "")
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assert.Equal(t, http.StatusBadRequest, response.Code)
		assertErrorMessage(t, urlIsEmptyMessage, response)
	})

	t.Run("request json is invalid", func(t *testing.T) {
		testStore := NewTestStore(baseURL)
		sut := New(testStore, baseURL)
		request := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader("{{]}"))
		request.Header.Set(contentTypeHeader, applicationJSON)
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assert.Equal(t, http.StatusBadRequest, response.Code)
		assertErrorMessage(t, "failed to parse request", response)
	})
}

func TestServeHTTP(t *testing.T) {
	t.Run("put method not allowed", func(t *testing.T) {
		want := http.StatusMethodNotAllowed
		testStore := NewTestStore(baseURL)
		sut := New(testStore, baseURL)
		request := newPutRequest(testURL)
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assert.Equal(t, want, response.Code)
	})
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

func newAPIShortenRequest(t *testing.T, url string) *http.Request {
	t.Helper()
	req := ShortenRequest{URL: url}
	body, err := json.Marshal(&req)

	require.NoError(t, err, "unable to marshal struct %q, '%v'", req, err)

	r := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(string(body)))
	r.Header.Set(contentTypeHeader, applicationJSON)
	return r
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

func getShortURLFromResponse(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()
	var r ShortenResponse
	err := json.NewDecoder(response.Body).Decode(&r)

	require.NoError(t, err, "unable to parse response from server %q, '%v'", response.Body, err)

	return r.Result
}
