package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type want struct {
	location string
	code     int
	body     string
}

type testURLStore struct {
	m map[string]string
}

func (t *testURLStore) Get(shortURL string) (string, error) {
	return t.m[shortURL], nil
}

func (t *testURLStore) Add(url string, shorten ShortenFunc) (string, error) {
	shortURL := shorten(uint32(len(t.m)))
	t.m[shortURL] = url
	return shortURL, nil
}


func TestGet(t *testing.T) {
	t.Run("get url", func(t *testing.T) {
		want := want{
			code:     http.StatusTemporaryRedirect,
			location: "https://practicum.yandex.ru/",
		}
		sut := ShortenerServer{
			store: &testURLStore{
				m: map[string]string{
					"EwHXdJfB": "https://practicum.yandex.ru/",
				},
			},
		}
		request := newGetRequest("EwHXdJfB")
		response := httptest.NewRecorder()

		sut.get(response, request)

		assertResponse(t, want, response)
	})

	t.Run("url not found by shortened url", func(t *testing.T) {
		want := want{
			code:     http.StatusBadRequest,
			location: "",
			body:     "shortened URL not found",
		}
		request := newGetRequest("something")
		response := httptest.NewRecorder()
		sut := ShortenerServer{
			store: &testURLStore{
				m: map[string]string{},
			},
		}

		sut.get(response, request)

		assertResponse(t, want, response)
	})

	t.Run("shortened url is empty", func(t *testing.T) {
		want := want{
			code:     http.StatusBadRequest,
			location: "",
			body:     "shortened URL is empty",
		}
		request := newGetRequest("")
		response := httptest.NewRecorder()
		sut := ShortenerServer{
			store: &testURLStore{
				m: map[string]string{},
			},
		}

		sut.get(response, request)

		assertResponse(t, want, response)
	})

	t.Run("content type is not textplain", func(t *testing.T) {
		want := want{
			code:     http.StatusBadRequest,
			location: "",
			body:     "content type is not text/plain",
		}
		request := httptest.NewRequest(http.MethodGet, "/123", nil)
		request.Header.Set(contentTypeHeader, "application/json")
		response := httptest.NewRecorder()
		sut := ShortenerServer{
			store: &testURLStore{
				m: map[string]string{
					"123": "abc.com",
				},
			},
		}

		sut.get(response, request)

		assertResponse(t, want, response)
	})
}

func TestShorten(t *testing.T) {
	t.Run("shorten url", func(t *testing.T) {
		want := want{
			code: http.StatusCreated,
			body: "http://localhost:8080/y",
		}
		sut := ShortenerServer{
			store: &testURLStore{
				m: map[string]string{},
			},
		}
		request := newShortenRequest("https://practicum.yandex.ru/")
		response := httptest.NewRecorder()

		sut.shorten(response, request)

		assertResponse(t, want, response)
	})

	t.Run("content type is not textplain", func(t *testing.T) {
		want := want{
			code: http.StatusBadRequest,
			body: "content type is not text/plain",
		}
		sut := ShortenerServer{
			store: &testURLStore{
				m: map[string]string{},
			},
		}
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://practicum.yandex.ru/"))
		request.Header.Set(contentTypeHeader, "application/json")
		response := httptest.NewRecorder()

		sut.shorten(response, request)

		assertResponse(t, want, response)
	})

	t.Run("shorten same url twice", func(t *testing.T) {
		want := want{
			code: http.StatusCreated,
			body: "http://localhost:8080/y",
		}
		sut := ShortenerServer{
			store: &testURLStore{
				m: map[string]string{},
			},
		}
		request := newShortenRequest("https://practicum.yandex.ru/")
		response := httptest.NewRecorder()

		//1
		sut.shorten(response, request)

		assertResponse(t, want, response)

		want.body = "http://localhost:8080/n"
		request = newShortenRequest("https://practicum.yandex.ru/")
		response = httptest.NewRecorder()

		//2
		sut.shorten(response, request)

		assertResponse(t, want, response)
	})

	t.Run("url is empty", func(t *testing.T) {
		want := want{
			code: http.StatusBadRequest,
			body: "url is empty",
		}
		sut := ShortenerServer{
			store: &testURLStore{
				m: map[string]string{},
			},
		}
		request := newShortenRequest("")
		response := httptest.NewRecorder()

		sut.shorten(response, request)

		assertResponse(t, want, response)
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

func assertResponse(t *testing.T, want want, got *httptest.ResponseRecorder) {
	t.Helper()
	assert.Equal(t, want.code, got.Code)
	assert.Equal(t, want.location, got.Header().Get(locationHeader))
	if want.body != "" {
		assert.Equal(t, want.body, strings.TrimSuffix(got.Body.String(), "\n"))
	}
}
