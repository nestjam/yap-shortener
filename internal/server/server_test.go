package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/nestjam/yap-shortener/internal/model"
	"github.com/nestjam/yap-shortener/internal/store"
	"github.com/stretchr/testify/assert"
)

const testURL = "https://practicum.yandex.ru/"

type want struct {
	location string
	code     int
	body     string
}

type testURLStore struct {
	m            map[string]string
	lastShortURL string
}

func NewTestStore() *testURLStore {
	return &testURLStore{
		m: map[string]string{},
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
	t.lastShortURL = shortURL
}

func TestRedirect(t *testing.T) {
	t.Run("get url", func(t *testing.T) {
		want := want{
			code:     http.StatusTemporaryRedirect,
			location: testURL,
			body: "<a href=\"https://practicum.yandex.ru/\">Temporary Redirect</a>.\n\n",
		}
		testStore := NewTestStore()
		testStore.m["EwHXdJfB"] = testURL
		sut := ShortenerServer{
			store: testStore,
		}
		request := newGetRequest("EwHXdJfB")
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assertGetResponse(t, want, response)
	})

	t.Run("shortened url is empty", func(t *testing.T) {
		want := want{
			code:     http.StatusBadRequest,
			location: "",
			body:     "shortened URL is empty\n",
		}
		request := newGetRequest("")
		response := httptest.NewRecorder()
		sut := ShortenerServer{
			store: NewTestStore(),
		}

		sut.ServeHTTP(response, request)

		assertGetResponse(t, want, response)
	})

	t.Run("url not found", func(t *testing.T) {
		want := want{
			code: http.StatusNotFound,
			body: "not found\n",
		}
		testStore := NewTestStore()
		sut := ShortenerServer{
			store: testStore,
		}
		request := newGetRequest("EwHXdJfB")
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assertGetResponse(t, want, response)
	})

	t.Run("url not found (in-memory store)", func(t *testing.T) {
		want := want{
			code: http.StatusNotFound,
			body: "not found\n",
		}
		testStore := store.NewInMemory()
		sut := ShortenerServer{
			store: testStore,
		}
		request := newGetRequest("EwHXdJfB")
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assertGetResponse(t, want, response)
	})

	// t.Run("content type is not textplain", func(t *testing.T) {
	// 	want := want{
	// 		code:     http.StatusBadRequest,
	// 		location: "",
	// 		body:     "content type is not text/plain",
	// 	}
	// 	request := httptest.NewRequest(http.MethodGet, "/123", nil)
	// 	request.Header.Set(contentTypeHeader, "application/json")
	// 	response := httptest.NewRecorder()
	// 	sut := ShortenerServer{
	// 		store: &testURLStore{
	// 			m: map[string]string{
	// 				"123": "abc.com",
	// 			},
	// 		},
	// 	}

	// 	sut.ServeHTTP(response, request)

	// 	assertGetResponse(t, want, response)
	// })
}

func TestShorten(t *testing.T) {
	t.Run("shorten url", func(t *testing.T) {
		want := want{
			code: http.StatusCreated,
		}
		testStore := NewTestStore()
		sut := ShortenerServer{
			store: testStore,
		}
		request := newShortenRequest(testURL)
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assert.Equal(t, want.code, response.Code)
		assertShortenedURL(t, testStore.lastShortURL, response)
	})

	t.Run("content type is not textplain", func(t *testing.T) {
		want := want{
			code: http.StatusBadRequest,
			body: "content type is not text/plain\n",
		}
		sut := ShortenerServer{
			store: NewTestStore(),
		}
		request := newShortenRequest(testURL)
		request.Header.Set(contentTypeHeader, "application/json")
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assert.Equal(t, want.code, response.Code)
		assertResponseBody(t, want.body, response)
	})

	t.Run("shorten same url twice", func(t *testing.T) {
		want := want{
			code: http.StatusCreated,
		}
		testStore := NewTestStore()
		sut := ShortenerServer{
			store: testStore,
		}
		request := newShortenRequest(testURL)
		response := httptest.NewRecorder()

		//1
		sut.ServeHTTP(response, request)

		assert.Equal(t, want.code, response.Code)
		assertShortenedURL(t, testStore.lastShortURL, response)

		request = newShortenRequest(testURL)
		response = httptest.NewRecorder()

		//2
		sut.ServeHTTP(response, request)

		assert.Equal(t, want.code, response.Code)
		assertShortenedURL(t, testStore.lastShortURL, response)
	})

	t.Run("url is empty", func(t *testing.T) {
		want := want{
			code: http.StatusBadRequest,
			body: "url is empty\n",
		}
		sut := ShortenerServer{
			store: NewTestStore(),
		}
		request := newShortenRequest("")
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assert.Equal(t, want.code, response.Code)
		assertResponseBody(t, want.body, response)
	})
}

func TestServeHTTP(t *testing.T) {
	t.Run("put method not supported", func(t *testing.T) {
		want := http.StatusBadRequest
		sut := ShortenerServer{
			store: NewTestStore(),
		}
		request := newPutRequest(testURL)
		response := httptest.NewRecorder()

		sut.ServeHTTP(response, request)

		assert.Equal(t, want, response.Code)
	})
}

func newGetRequest(shortURL string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/" + shortURL, nil)
	r.Header.Set(contentTypeHeader, "text/plain; charset=utf-8")
	return r
}

func newShortenRequest(url string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(url))
	r.Header.Set(contentTypeHeader, "text/plain; charset=utf-8")
	return r
}

func newPutRequest(url string) *http.Request {
	r := httptest.NewRequest(http.MethodPut, "/", strings.NewReader(url))
	r.Header.Set(contentTypeHeader, "text/plain; charset=utf-8")
	return r
}

func assertShortenedURL(t *testing.T, stored string, got *httptest.ResponseRecorder) {
	t.Helper()
	want := domain + "/" + stored
	assert.Equal(t, want, got.Body.String())
}

func assertGetResponse(t *testing.T, want want, got *httptest.ResponseRecorder) {
	t.Helper()
	assert.Equal(t, want.code, got.Code)
	assert.Equal(t, want.location, got.Header().Get(locationHeader))
	assert.Equal(t, want.body, got.Body.String())
}

func assertResponseBody(t *testing.T, want string, got *httptest.ResponseRecorder){
	t.Helper()
	assert.Equal(t, want, got.Body.String())
}
