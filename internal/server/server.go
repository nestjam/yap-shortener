package server

import (
	"io"
	"net/http"
	"strings"

	"github.com/nestjam/yap-shortener/internal/shortener"
)

const (
	locationHeader    = "Location"
	contentTypeHeader = "Content-Type"
	textPlain         = "text/plain"
	host = "http://localhost:8080"
)

type ShortenFunc func(id uint32) string

type UrlStore interface {
	Get(shortUrl string) (string, error)

	Add(url string, shorten ShortenFunc) (string, error)
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
		BadRequest(w,  "")
	}
}

func (s *ShortenerServer) get(w http.ResponseWriter, r *http.Request) {
	var contentType = r.Header.Get(contentTypeHeader)
	if contentType != textPlain {
		BadRequest(w,  "content type is not text/plain")
		return
	}

	var path = strings.TrimPrefix(r.URL.Path, "/")
	if len(path) == 0 {
		BadRequest(w,  "shortened URL is empty")
		return
	}

	url, err := s.store.Get(path)

	if err != nil {
		BadRequest(w,  err.Error())
		return
	}

	if len(url) == 0 {
		BadRequest(w,  "shortened URL not found")
		return
	}

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (s *ShortenerServer) shorten(w http.ResponseWriter, r *http.Request) {
	var contentType = r.Header.Get(contentTypeHeader)
	if contentType != textPlain {
		BadRequest(w,  "content type is not text/plain")
		return
	}
	
	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)

	if err != nil {
		BadRequest(w, err.Error())
		return
	}

	if len(body) == 0 {
		BadRequest(w, "url is empty")
		return
	}

	shortUrl, err := s.store.Add(string(body), shortener.Shorten)

	if err != nil {
		BadRequest(w, err.Error())
		return
	}

	w.Header().Set(contentTypeHeader, textPlain)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(host + "/" + shortUrl))
}

func BadRequest(w http.ResponseWriter, err string){
	http.Error(w, err, http.StatusBadRequest)
}
