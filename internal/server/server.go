package server

import (
	"io"
	"mime"
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

type URLStore interface {
	Get(shortURL string) (string, error)

	//TODO: Вынести генерацию id в отдельный тип или функцию вне UrlStore
	Add(url string, shorten ShortenFunc) (string, error)
}

type ShortenerServer struct {
	store URLStore
}

func New(store URLStore) *ShortenerServer {
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
		BadRequest(w,  "bad request")
	}
}

func (s *ShortenerServer) get(w http.ResponseWriter, r *http.Request) {
	// if !HasContentType(r, textPlain) {
	// 	BadRequest(w,  "content type is not text/plain")
	// 	return
	// }

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
	if !HasContentType(r, textPlain) {
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

func HasContentType(r *http.Request, mimetype string) bool {
	contentType := r.Header.Get("Content-type")
	if contentType == "" {
		return mimetype == "application/octet-stream"
	}

	for _, v := range strings.Split(contentType, ",") {
		t, _, err := mime.ParseMediaType(v)
		if err != nil {
			break
		}
		if t == mimetype {
			return true
		}
	}
	return false
}
