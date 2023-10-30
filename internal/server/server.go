package server

import (
	"net/http"
	"strings"
)

const (
	locationHeader    = "Location"
	contentTypeHeader = "Content-Type"
	textPlain         = "text/plain"
)

type UrlStore interface {
	Get(shortUrl string) string
}

type ShortenerServer struct {
	store UrlStore
}

func New(store UrlStore) *ShortenerServer {
	return &ShortenerServer{
		store,
	}
}

func (s *ShortenerServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.get(w, r)
	} else if r.Method == http.MethodPost {
		s.shorten(w, r)
	} else {
		http.Error(w, "", http.StatusBadRequest)
	}
}

func (s *ShortenerServer) get(w http.ResponseWriter, r *http.Request) {
	var contentType = r.Header.Get(contentTypeHeader)
	if contentType != textPlain {
		http.Error(w, "content type is not text/plain", http.StatusBadRequest)
		return
	}

	var path = strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		http.Error(w, "shortened URL is empty", http.StatusBadRequest)
		return
	}

	url := s.store.Get(path)
	if url == "" {
		http.Error(w, "shortened URL not found", http.StatusBadRequest)
		return
	}

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (s *ShortenerServer) shorten(w http.ResponseWriter, r *http.Request) {
	var contentType = r.Header.Get(contentTypeHeader)
	if contentType != textPlain {
		http.Error(w, "content type is not text/plain", http.StatusBadRequest)
		return
	}

	w.Header().Set(contentTypeHeader, textPlain)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("http://localhost:8080/EwHXdJfB"))
}
