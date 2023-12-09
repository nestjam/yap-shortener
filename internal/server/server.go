package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	chimiddleware "github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/nestjam/yap-shortener/internal/domain"
	"github.com/nestjam/yap-shortener/internal/middleware"
	"github.com/nestjam/yap-shortener/internal/shortener"
	"go.uber.org/zap"
)

const (
	locationHeader                 = "Location"
	contentTypeHeader              = "Content-Type"
	contentLengthHeader            = "Content-Length"
	textPlain                      = "text/plain"
	applicationJSON                = "application/json"
	applicationGZIP                = "application/x-gzip"
	urlIsEmptyMessage              = "url is empty"
	batchIsEmptyMessage            = "batch is empty"
	failedToWriterResponseMessage  = "failed to prepare response"
	failedToStoreURLMessage        = "failed to store url"
	failedToParseRequestMessage    = "failed to parse request"
	failedToPrepareResponseMessage = "failed to prepare response"
)

type Server struct {
	storage domain.URLStore
	router  chi.Router
	baseURL string
}

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	Result string `json:"result"`
}

type OriginalURL struct {
	CorrelationID string `json:"correlation_id"`
	URL           string `json:"original_url"`
}

type ShortURL struct {
	CorrelationID string `json:"correlation_id"`
	URL           string `json:"short_url"`
}

func New(storage domain.URLStore, baseURL string, logger *zap.Logger) *Server {
	r := chi.NewRouter()
	s := &Server{
		storage,
		r,
		baseURL,
	}

	r.Use(middleware.ResponseLogger(logger))

	r.Group(func(r chi.Router) {
		r.Get("/ping", s.ping)
	})

	r.Group(func(r chi.Router) {
		r.Use(chimiddleware.AllowContentType(applicationJSON))
		r.Use(middleware.RequestDecoder, middleware.ResponseEncoder)

		r.Post("/api/shorten/batch", s.shortenBatchAPI)
		r.Post("/api/shorten", s.shortenAPI)
	})

	r.Group(func(r chi.Router) {
		r.Use(chimiddleware.AllowContentType(textPlain, applicationGZIP))
		r.Use(middleware.RequestDecoder, middleware.ResponseEncoder)

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
	url, err := s.storage.Get(r.Context(), key)

	if errors.Is(err, domain.ErrOriginalURLNotFound) {
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
	err = s.storage.Add(r.Context(), shortURL, string(body))

	var originalURLAlreadyExists *domain.OriginalURLExistsError
	if err != nil && !errors.As(err, &originalURLAlreadyExists) {
		internalError(w, failedToStoreURLMessage)
		return
	}

	status := http.StatusCreated
	if originalURLAlreadyExists != nil {
		status = http.StatusConflict
		shortURL = originalURLAlreadyExists.GetShortURL()
	}
	w.Header().Set(contentTypeHeader, textPlain)
	w.WriteHeader(status)
	_, err = w.Write([]byte(joinPath(s.baseURL, shortURL)))

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
		badRequest(w, failedToParseRequestMessage)
		return
	}

	if len(req.URL) == 0 {
		badRequest(w, urlIsEmptyMessage)
		return
	}

	shortURL := shortener.Shorten(uuid.New().ID())
	err = s.storage.Add(r.Context(), shortURL, req.URL)

	var originalURLAlreadyExists *domain.OriginalURLExistsError
	if err != nil && !errors.As(err, &originalURLAlreadyExists) {
		internalError(w, failedToStoreURLMessage)
		return
	}

	status := http.StatusCreated
	if originalURLAlreadyExists != nil {
		status = http.StatusConflict
		shortURL = originalURLAlreadyExists.GetShortURL()
	}

	resp := ShortenResponse{Result: joinPath(s.baseURL, shortURL)}
	content, err := json.Marshal(resp)

	if err != nil {
		internalError(w, failedToPrepareResponseMessage)
		return
	}

	w.Header().Set(contentTypeHeader, applicationJSON)
	w.Header().Set(contentLengthHeader, strconv.Itoa(len(content)))
	w.WriteHeader(status)
	_, err = w.Write(content)

	if err != nil {
		internalError(w, failedToWriterResponseMessage)
		return
	}
}

func (s *Server) ping(w http.ResponseWriter, r *http.Request) {
	status := http.StatusInternalServerError
	if s.storage.IsAvailable(r.Context()) {
		status = http.StatusOK
	}
	w.WriteHeader(status)
}

func (s *Server) shortenBatchAPI(w http.ResponseWriter, r *http.Request) {
	var req []OriginalURL
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&req)

	if err != nil {
		badRequest(w, failedToParseRequestMessage)
		return
	}

	if len(req) == 0 {
		badRequest(w, batchIsEmptyMessage)
		return
	}

	urlPairs := make([]domain.URLPair, len(req))
	for i := 0; i < len(req); i++ {
		urlPairs[i] = domain.URLPair{
			ShortURL:    shortener.Shorten(uuid.New().ID()),
			OriginalURL: req[i].URL,
		}
	}

	err = s.storage.AddBatch(r.Context(), urlPairs)

	if err != nil {
		internalError(w, failedToStoreURLMessage)
		return
	}

	resp := make([]ShortURL, len(req))
	for i := 0; i < len(req); i++ {
		resp[i] = ShortURL{
			CorrelationID: req[i].CorrelationID,
			URL:           joinPath(s.baseURL, urlPairs[i].ShortURL),
		}
	}

	content, err := json.Marshal(resp)

	if err != nil {
		internalError(w, failedToPrepareResponseMessage)
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

func joinPath(base, elem string) string {
	return fmt.Sprintf("%s/%s", base, elem)
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
