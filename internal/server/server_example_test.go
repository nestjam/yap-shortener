package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/nestjam/yap-shortener/internal/domain"
	"github.com/nestjam/yap-shortener/internal/persistance/inmemory"
)

func ExampleServer_ServeHTTP_expandURL() {
	u := URLShortenerTest{
		CreateDependencies: func() (domain.URLStore, Cleanup) {
			return inmemory.New(), func() {}
		},
	}
	urlStore, cleanup := u.CreateDependencies()
	defer cleanup()
	userID := domain.NewUserID()
	pair := domain.URLPair{
		ShortURL:    "EwHXdJfB",
		OriginalURL: "https://practicum.yandex.ru/",
	}
	_ = urlStore.AddURL(context.Background(), pair, userID)
	sut := New(urlStore, baseURL)

	request := httptest.NewRequest(http.MethodGet, "/"+pair.ShortURL, nil)
	request.Header.Set(contentTypeHeader, textPlain+"; charset=utf-8")

	response := httptest.NewRecorder()

	sut.ServeHTTP(response, request)

	fmt.Println(response.Code)
	fmt.Println(response.Header().Get("Location"))

	// Output:
	// 307
	// https://practicum.yandex.ru/
}
