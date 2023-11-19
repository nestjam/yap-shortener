package server

import (
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nestjam/yap-shortener/internal/log"
	"github.com/nestjam/yap-shortener/internal/model"
	"github.com/nestjam/yap-shortener/internal/shortener"
)

const (
	locationHeader    = "Location"
	contentTypeHeader = "Content-Type"
	textPlain         = "text/plain"
)

type URLStore interface {
	Get(shortURL string) (string, error)

	Add(shortURL, url string)
}

type Server struct {
	store   URLStore
	router  chi.Router
	baseURL string
}

func New(store URLStore, baseURL string) *Server {
	r := chi.NewRouter()
	s := &Server{
		store,
		r,
		baseURL,
	}

	r.Use(log.RequestResponseLogger)
	r.Use(middleware.AllowContentType("text/plain"))

	r.Get("/{key}", s.redirect)
	r.Post("/", s.shorten)

	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) redirect(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	url, err := s.store.Get(key)

	if errors.Is(err, model.ErrNotFound) {
		notFound(w, err.Error())
		return
	}

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (s *Server) shorten(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	_ = r.Body.Close()

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
	_, err = w.Write([]byte(s.baseURL + "/" + shortURL))

	if err != nil {
		internalError(w, "failed to write response")
		return
	}
}

func badRequest(w http.ResponseWriter, err string) {
	http.Error(w, err, http.StatusBadRequest)
}

func notFound(w http.ResponseWriter, err string) {
	http.Error(w, err, http.StatusNotFound)
}

func internalError(w http.ResponseWriter, err string) {
	http.Error(w, err, http.StatusInternalServerError)
}
