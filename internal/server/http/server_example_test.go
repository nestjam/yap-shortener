package server

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/nestjam/yap-shortener/internal/persistance/inmemory"
)

func ExampleServer_ServeHTTP() {
	const baseURL = "http://localhost:8080"
	store := inmemory.New()
	sut := New(store, baseURL)

	const (
		longURL           = "https://practicum.yandex.ru/"
		contentType       = "text/plain; charset=utf-8"
		contentTypeHeader = "Content-Type"
	)
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(longURL))
	request.Header.Set(contentTypeHeader, contentType)
	response := httptest.NewRecorder()

	sut.ServeHTTP(response, request)

	fmt.Println(response.Code)
	shortURL := response.Body.String()

	request = httptest.NewRequest(http.MethodGet, shortURL, nil)
	request.Header.Set(contentTypeHeader, contentType)
	response = httptest.NewRecorder()

	sut.ServeHTTP(response, request)

	fmt.Println(response.Code)
	fmt.Println(response.Header().Get("Location"))

	// Output:
	// 201
	// 307
	// https://practicum.yandex.ru/
}
