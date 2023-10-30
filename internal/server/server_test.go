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

type testUrlStore struct {
	m map[string]string
}

func (t *testUrlStore) Get(shortUrl string) string {
	return t.m[shortUrl]
}

func TestGet(t *testing.T) {
	t.Run("get url", func(t *testing.T) {
		want := want{
			code:     http.StatusTemporaryRedirect,
			location: "https://practicum.yandex.ru/",
		}
		sut := ShortenerServer{
			store: &testUrlStore{
				m: map[string]string{
					"EwHXdJfB": "https://practicum.yandex.ru/",
				},
			},
		}
		request := httptest.NewRequest(http.MethodGet, "/EwHXdJfB", nil)
		request.Header.Set(contentTypeHeader, textPlain)
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
		request := httptest.NewRequest(http.MethodGet, "/something", nil)
		request.Header.Set(contentTypeHeader, textPlain)
		response := httptest.NewRecorder()
		sut := ShortenerServer{
			store: &testUrlStore{
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
		request := httptest.NewRequest(http.MethodGet, "/", nil)
		request.Header.Set(contentTypeHeader, textPlain)
		response := httptest.NewRecorder()
		sut := ShortenerServer{
			store: &testUrlStore{
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
			store: &testUrlStore{
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
			body: "http://localhost:8080/EwHXdJfB",
		}
		sut := ShortenerServer{
			store: &testUrlStore{
				m: map[string]string{},
			},
		}
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://practicum.yandex.ru/"))
		request.Header.Set(contentTypeHeader, textPlain)
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
			store: &testUrlStore{
				m: map[string]string{},
			},
		}
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://practicum.yandex.ru/"))
		request.Header.Set(contentTypeHeader, "application/json")
		response := httptest.NewRecorder()

		sut.shorten(response, request)

		assertResponse(t, want, response)
	})
}

func assertResponse(t *testing.T, want want, got *httptest.ResponseRecorder) {
	t.Helper()
	assert.Equal(t, want.code, got.Code)
	assert.Equal(t, want.location, got.Header().Get(locationHeader))
	if want.body != "" {
		assert.Equal(t, want.body, strings.TrimSuffix(got.Body.String(), "\n"))
	}
}
