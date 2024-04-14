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
	"go.uber.org/zap"

	"github.com/nestjam/yap-shortener/internal/auth"
	customctx "github.com/nestjam/yap-shortener/internal/context"
	"github.com/nestjam/yap-shortener/internal/domain"
	"github.com/nestjam/yap-shortener/internal/domain/service"
	"github.com/nestjam/yap-shortener/internal/middleware"
	"github.com/nestjam/yap-shortener/internal/shortener"
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
	failedToWriteResponseMessage   = "failed to write response"
	failedToStoreURLMessage        = "failed to store url"
	failedToParseRequestMessage    = "failed to parse request"
	failedToPrepareResponseMessage = "failed to prepare response"
	failedToParseCIDRMessage       = "failed to parse trusted subnet address"
)

// Server предоставляет возможность сокращать URL, получать исходный и управлять сокращенными URL.
type Server struct {
	store               domain.URLStore
	router              chi.Router
	logger              *zap.Logger
	urlRemover          *service.URLRemover
	baseURL             string
	trustedSubnet       string
	shortenURLsMaxCount int
}

// ShortenRequest представляет тело запроса и содержит исходный URL.
type ShortenRequest struct {
	URL string `json:"url"` // исходный URL
}

// ShortenResponse содержит сокращенный URL.
type ShortenResponse struct {
	Result string `json:"result"` // сокращенный URL
}

// OriginalURL содержит исходный URL. Применяется в запросе сокращения набора URL.
type OriginalURL struct {
	CorrelationID string `json:"correlation_id"` // идентификатор для сопоставления исходного и сокращенного URL
	URL           string `json:"original_url"`   // исходный URL
}

// ShortURL содержит сокращенный URL. Возвращается в ответе на запрос сокращения набора URL.
type ShortURL struct {
	CorrelationID string `json:"correlation_id"` // идентификатор для сопоставления исходного и сокращенного URL
	URL           string `json:"short_url"`      // сокращенный URL
}

// UserURL содержит исходный и сокращенный URL. Возвращается в ответе на запрос набора URL, сокращенного пользователем.
type UserURL struct {
	ShortURL    string `json:"short_url"`    // сокращенный URL
	OriginalURL string `json:"original_url"` // исходный URL
}

// Stats содержит количество сокращенных URL и количество пользователей в сервисе.
type Stats struct {
	URLs  int `json:"urls"`  // количество сокращённых URL в сервисе
	Users int `json:"users"` // количество пользователей в сервисе
}

// Option определяет опцию настройки сервера.
type Option func(*Server)

// New создает сервер. Конструктор принимает на вход хранилище URL, базовый URL и набор опций.
func New(store domain.URLStore, baseURL string, options ...Option) *Server {
	r := chi.NewRouter()
	s := &Server{
		store:   store,
		router:  r,
		baseURL: baseURL,
		logger:  zap.NewNop(),
	}

	for _, opt := range options {
		opt(s)
	}

	authorizer := auth.New(auth.SecretKey, auth.TokenExp)
	const apiUserURLsPath = "/api/user/urls"

	r.Use(middleware.ResponseLogger(s.logger))

	r.Group(func(r chi.Router) {
		r.Get("/ping", s.ping)
	})

	r.Group(func(r chi.Router) {
		r.Use(chimiddleware.AllowContentType(applicationJSON))
		r.Use(middleware.RequestDecoder, middleware.ResponseEncoder)
		r.Use(middleware.Auth(authorizer))

		r.Post("/api/shorten/batch", s.shortenURLs)
		r.Post("/api/shorten", s.shortenAPI)

		r.Delete(apiUserURLsPath, s.deleteUserURLs)
	})

	r.Group(func(r chi.Router) {
		r.Use(chimiddleware.AllowContentType(textPlain, applicationGZIP))
		r.Use(middleware.RequestDecoder, middleware.ResponseEncoder)

		r.Get("/{key}", s.redirect)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(authorizer))

			r.Post("/", s.shorten)
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.ResponseEncoder)
		r.Use(middleware.Auth(authorizer))

		r.Get(apiUserURLsPath, s.getUserURLs)
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.TrustedSubnet(s.trustedSubnet))

		r.Get("/api/internal/stats", s.getStats)
	})

	return s
}

// ServeHTTP обрабатывает запрос.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

func (s *Server) redirect(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	ctx := r.Context()
	url, err := s.store.GetOriginalURL(ctx, key)

	if errors.Is(err, domain.ErrOriginalURLNotFound) {
		notFound(w, err.Error())
		return
	}

	if errors.Is(err, domain.ErrOriginalURLIsDeleted) {
		http.Error(w, "url is deleted", http.StatusGone)
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

	ctx := r.Context()
	user, _ := customctx.GetUser(ctx)
	shortURL := shortener.Shorten(uuid.New().ID())
	pair := domain.URLPair{
		ShortURL:    shortURL,
		OriginalURL: string(body),
	}
	err = s.store.AddURL(ctx, pair, user.ID)

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
		internalError(w, failedToWriteResponseMessage)
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

	ctx := r.Context()
	user, _ := customctx.GetUser(ctx)
	shortURL := shortener.Shorten(uuid.New().ID())
	pair := domain.URLPair{
		ShortURL:    shortURL,
		OriginalURL: req.URL,
	}
	err = s.store.AddURL(ctx, pair, user.ID)

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
		internalError(w, failedToWriteResponseMessage)
		return
	}
}

func (s *Server) ping(w http.ResponseWriter, r *http.Request) {
	status := http.StatusInternalServerError
	ctx := r.Context()
	if s.store.IsAvailable(ctx) {
		status = http.StatusOK
	}
	w.WriteHeader(status)
}

func (s *Server) shortenURLs(w http.ResponseWriter, r *http.Request) {
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

	if isTooManyURLs(req, s.shortenURLsMaxCount) {
		forbidden(w)
		return
	}

	urlPairs := make([]domain.URLPair, len(req))
	for i := 0; i < len(req); i++ {
		if len(req[i].URL) == 0 {
			badRequest(w, urlIsEmptyMessage)
			return
		}

		urlPairs[i] = domain.URLPair{
			ShortURL:    shortener.Shorten(uuid.New().ID()),
			OriginalURL: req[i].URL,
		}
	}

	ctx := r.Context()
	user, _ := customctx.GetUser(ctx)
	err = s.store.AddURLs(ctx, urlPairs, user.ID)

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
		internalError(w, failedToWriteResponseMessage)
		return
	}
}

func (s *Server) getUserURLs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, _ := customctx.GetUser(ctx)

	if user.IsNew {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	urlPairs, _ := s.store.GetUserURLs(ctx, user.ID)

	if len(urlPairs) == 0 {
		http.Error(w, "no urls", http.StatusNoContent)
		return
	}

	resp := make([]UserURL, len(urlPairs))
	for i := 0; i < len(urlPairs); i++ {
		resp[i] = UserURL{
			OriginalURL: urlPairs[i].OriginalURL,
			ShortURL:    joinPath(s.baseURL, urlPairs[i].ShortURL),
		}
	}
	content, _ := json.Marshal(resp)

	w.Header().Set(contentTypeHeader, applicationJSON)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}

func (s *Server) deleteUserURLs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, _ := customctx.GetUser(ctx)

	var shortURLs []string
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&shortURLs)

	if err != nil {
		badRequest(w, failedToParseRequestMessage)
		return
	}

	if s.urlRemover != nil {
		err = s.urlRemover.DeleteURLs(shortURLs, user.ID)
	} else {
		err = s.store.DeleteUserURLs(ctx, shortURLs, user.ID)
	}

	if err != nil {
		internalError(w, "failed to delete user urls")
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func (s *Server) getStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var err error
	stats := Stats{}
	stats.URLs, stats.Users, err = s.store.GetURLsAndUsersCount(ctx)
	if err != nil {
		internalError(w, "failed to get stats")
		return
	}

	content, err := json.Marshal(stats)
	if err != nil {
		internalError(w, failedToPrepareResponseMessage)
		return
	}

	w.WriteHeader(http.StatusOK)
	_, err = w.Write(content)
	if err != nil {
		internalError(w, failedToWriteResponseMessage)
		return
	}
}

func isTooManyURLs(urls []OriginalURL, maxCount int) bool {
	return maxCount > 0 && len(urls) > maxCount
}

func forbidden(w http.ResponseWriter) {
	http.Error(w, "too many urls", http.StatusForbidden)
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

// WithLogger задает логер для сервера.
func WithLogger(logger *zap.Logger) Option {
	return func(s *Server) {
		s.logger = logger
	}
}

// WithShortenURLsMaxCount определяет максимальное количество URL в запросе на сокращение коллекции URL.
func WithShortenURLsMaxCount(count int) Option {
	return func(s *Server) {
		s.shortenURLsMaxCount = count
	}
}

// WithURLsRemover задает компонент, который выполняет удаление сохраненных URL.
func WithURLsRemover(remover *service.URLRemover) Option {
	return func(s *Server) {
		s.urlRemover = remover
	}
}

// WithTrustedSubnet задает доверенную подсеть.
func WithTrustedSubnet(subnet string) Option {
	return func(s *Server) {
		s.trustedSubnet = subnet
	}
}
