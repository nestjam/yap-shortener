package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nestjam/yap-shortener/internal/compress"
	"github.com/nestjam/yap-shortener/internal/log"
	"github.com/nestjam/yap-shortener/internal/model"
	"github.com/nestjam/yap-shortener/internal/shortener"
)

const (
	locationHeader                = "Location"
	contentTypeHeader             = "Content-Type"
	contentLengthHeader           = "Content-Length"
	textPlain                     = "text/plain"
	applicationJSON               = "application/json"
	applicationGZIP               = "application/x-gzip"
	urlIsEmptyMessage             = "url is empty"
	failedToWriterResponseMessage = "failed to prepare response"
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

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	Result string `json:"result"`
}

func New(store URLStore, baseURL string) *Server {
	r := chi.NewRouter()
	s := &Server{
		store,
		r,
		baseURL,
	}

	r.Use(log.RequestResponseLogger)

	r.Route("/api/shorten", func(r chi.Router) {
		r.Use(middleware.AllowContentType(applicationJSON))
		r.Use(compress.Compressor)
		r.Post("/", s.shortenAPI)
	})

	r.Route("/", func(r chi.Router) {
		r.Use(middleware.AllowContentType(textPlain, applicationGZIP))
		r.Use(compress.Compressor)
		r.Get("/{key}", s.redirect)
		r.Post("/", s.shorten)
	})

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
		badRequest(w, urlIsEmptyMessage)
		return
	}

	shortURL := shortener.Shorten(uuid.New().ID())
	s.store.Add(shortURL, string(body))

	w.Header().Set(contentTypeHeader, textPlain)
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write([]byte(s.baseURL + "/" + shortURL))

	if err != nil {
		internalError(w, failedToWriterResponseMessage)
		return
	}
}

func (s *Server) shortenAPI(w http.ResponseWriter, r *http.Request) {
	var req ShortenRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)

	if err != nil {
		badRequest(w, "failed to parse request")
		return
	}

	if len(req.URL) == 0 {
		badRequest(w, urlIsEmptyMessage)
		return
	}

	shortURL := shortener.Shorten(uuid.New().ID())
	s.store.Add(shortURL, req.URL)

	resp := ShortenResponse{Result: s.baseURL + "/" + shortURL}
	content, err := json.Marshal(resp)

	if err != nil {
		internalError(w, "failed to prepare response")
		return
	}

	w.Header().Set(contentTypeHeader, applicationJSON)
	w.Header().Set(contentLengthHeader, strconv.Itoa(len(content)))
	w.WriteHeader(http.StatusCreated)
	_, err = w.Write(content)

	if err != nil {
		internalError(w, failedToWriterResponseMessage)
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
