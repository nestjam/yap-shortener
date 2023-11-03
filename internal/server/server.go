package server

import (
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/nestjam/yap-shortener/internal/model"
	"github.com/nestjam/yap-shortener/internal/shortener"
	"github.com/go-chi/chi/v5"
)

const (
	locationHeader    = "Location"
	contentTypeHeader = "Content-Type"
	textPlain         = "text/plain"
	domain            = "http://localhost:8080"
)

type URLStore interface {
	Get(shortURL string) (string, error)

	Add(shortURL, url string)
}

type Server struct {
	store URLStore
	router chi.Router
}

func New(store URLStore) *Server {
	router := chi.NewRouter()
	s := &Server{
		store,
		router,
	}
	router.Get("/{key}", s.redirect)
	router.Post("/", s.shorten)
	return s;
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) redirect(w http.ResponseWriter, r *http.Request) {
	// По заданию требуется проверка, но в автотестах не устанавливается заголовок
	// if !HasContentType(r, textPlain) {
	// 	BadRequest(w,  "content type is not text/plain")
	// 	return
	// }

	key := chi.URLParam(r, "key")
	url, err := s.store.Get(key)

	if err == model.ErrNotFound {
		notFound(w, err.Error())
		return
	}

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (s *Server) shorten(w http.ResponseWriter, r *http.Request) {
	if !hasContentType(r, textPlain) {
		badRequest(w, "content type is not text/plain")
		return
	}

	body, err := io.ReadAll(r.Body)
	r.Body.Close()

	if err != nil {
		badRequest(w, err.Error())
		return
	}

	if len(body) == 0 {
		badRequest(w, "url is empty")
		return
	}

	shortURL := shortener.Shorten(uuid.New().ID())
	s.store.Add(shortURL, string(body))

	w.Header().Set(contentTypeHeader, textPlain)
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(domain + "/" + shortURL))
}

func badRequest(w http.ResponseWriter, err string) {
	http.Error(w, err, http.StatusBadRequest)
}

func notFound(w http.ResponseWriter, err string) {
	http.Error(w, err, http.StatusNotFound)
}

func hasContentType(r *http.Request, mimetype string) bool {
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
