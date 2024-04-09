package grpc

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/nestjam/yap-shortener/internal/auth"
	"github.com/nestjam/yap-shortener/internal/domain"
	"github.com/nestjam/yap-shortener/internal/server"
	"github.com/nestjam/yap-shortener/internal/shortener"
	pb "github.com/nestjam/yap-shortener/proto"

	customctx "github.com/nestjam/yap-shortener/internal/context"
)

const (
	urlIsEmptyMessage         = "url is empty"
	batchIsEmptyMessage       = "batch is empty"
	failedToStoreURLMessage   = "failed to store url"
	failedToGetURLsMessage    = "failed to get urls"
	tooManyURLsMessage        = "too many urls"
	unauthorizedMessage       = "unauthorized"
	failedToDeleteURLsMessage = "failed to delete user urls"
)

// Server предоставляет возможность сокращать ссылку, получать исходную и управлять ссылку.
type Server struct {
	pb.UnimplementedShortenerServer
	store               domain.URLStore
	urlRemover          *server.URLRemover
	userAuth            *auth.UserAuth
	baseURL             string
	shortenURLsMaxCount int
}

// Option определяет опцию настройки сервера.
type Option func(*Server)

// WithShortenURLsMaxCount определяет максимальное количество ссылок в запросе на сокращение коллекции ссылок.
func WithShortenURLsMaxCount(count int) Option {
	return func(s *Server) {
		s.shortenURLsMaxCount = count
	}
}

// WithURLsRemover задает компонент, который выполняет удаление сохраненных ссылок.
func WithURLsRemover(remover *server.URLRemover) Option {
	return func(s *Server) {
		s.urlRemover = remover
	}
}

// New создает сервер. Конструктор принимает на вход хранилище ссылок, базовую ссылку и набор опций.
func New(store domain.URLStore, baseURL string, options ...Option) *Server {
	s := &Server{
		store:    store,
		baseURL:  baseURL,
		userAuth: auth.New(auth.SecretKey, auth.TokenExp),
	}

	for _, opt := range options {
		opt(s)
	}

	return s
}

// Login выполняет авторизацию пользователя.
func (s *Server) Login(ctx context.Context, request *pb.LoginRequest) (*pb.LoginResponse, error) {
	// Сгенерируем новый ID, т.к. пользователи нигде не хранятся
	userID := domain.NewUserID()

	token, err := s.userAuth.BuildJWT(userID)
	if err != nil {
		return nil, errors.Wrap(err, "login")
	}

	return &pb.LoginResponse{Token: token}, nil
}

// Ping проверяет доступность сервиса.
func (s *Server) Ping(ctx context.Context, request *pb.PingRequest) (*pb.PingResponse, error) {
	response := &pb.PingResponse{
		Result: s.store.IsAvailable(ctx),
	}
	return response, nil
}

// GetOriginalURL возвращает исходную ссылку по ключу.
//
//nolint:lll // naturally long name
func (s *Server) GetOriginalURL(ctx context.Context, request *pb.GetOriginalURLRequest) (*pb.GetOriginalURLResponse, error) {
	const op = "get original url"
	url, err := s.store.GetOriginalURL(ctx, request.Key)

	if errors.Is(err, domain.ErrOriginalURLNotFound) ||
		errors.Is(err, domain.ErrOriginalURLIsDeleted) {
		return nil, errors.Wrap(status.Error(codes.NotFound, err.Error()), op)
	}

	response := &pb.GetOriginalURLResponse{
		OriginalURL: url,
	}
	return response, nil
}

// ShortenURL возвращает сокращенную ссылку.
func (s *Server) ShortenURL(ctx context.Context, request *pb.ShortenURLRequest) (*pb.ShortenURLResponse, error) {
	const op = "shorten url"

	if len(request.URL) == 0 {
		return nil, errors.Wrap(status.Error(codes.InvalidArgument, urlIsEmptyMessage), op)
	}

	user, _ := customctx.GetUser(ctx)
	shortURL := shortener.Shorten(uuid.New().ID())
	pair := domain.URLPair{
		ShortURL:    shortURL,
		OriginalURL: request.URL,
	}
	err := s.store.AddURL(ctx, pair, user.ID)

	var originalURLAlreadyExists *domain.OriginalURLExistsError
	if err != nil && !errors.As(err, &originalURLAlreadyExists) {
		return nil, errors.Wrap(status.Error(codes.Internal, failedToStoreURLMessage), op)
	}

	if originalURLAlreadyExists != nil {
		response := &pb.ShortenURLResponse{
			ShortenedURL: originalURLAlreadyExists.GetShortURL(),
		}
		return response, errors.Wrap(status.Error(codes.AlreadyExists, err.Error()), op)
	}

	response := &pb.ShortenURLResponse{
		ShortenedURL: shortURL,
	}
	return response, nil
}

// ShortenURLs сокращает переданные ссылки.
func (s *Server) ShortenURLs(ctx context.Context, request *pb.ShortenURLsRequest) (*pb.ShortenURLsResponse, error) {
	const op = "shorten urls"

	if len(request.URLs) == 0 {
		return nil, errors.Wrap(status.Error(codes.InvalidArgument, batchIsEmptyMessage), op)
	}

	if isTooManyURLs(request.URLs, s.shortenURLsMaxCount) {
		return nil, errors.Wrap(status.Error(codes.InvalidArgument, tooManyURLsMessage), op)
	}

	urlPairs := make([]domain.URLPair, len(request.URLs))
	for i := 0; i < len(request.URLs); i++ {
		originalURL := request.URLs[i].URL

		if len(originalURL) == 0 {
			return nil, errors.Wrap(status.Error(codes.InvalidArgument, urlIsEmptyMessage), op)
		}

		urlPairs[i] = domain.URLPair{
			ShortURL:    shortener.Shorten(uuid.New().ID()),
			OriginalURL: originalURL,
		}
	}

	user, _ := customctx.GetUser(ctx)
	err := s.store.AddURLs(ctx, urlPairs, user.ID)

	if err != nil {
		return nil, errors.Wrap(status.Error(codes.Internal, failedToStoreURLMessage), op)
	}

	shortenedURLs := make([]*pb.CorrelatedURL, len(request.URLs))
	for i := 0; i < len(request.URLs); i++ {
		shortenedURLs[i] = &pb.CorrelatedURL{
			CorrelationID: request.URLs[i].CorrelationID,
			URL:           joinPath(s.baseURL, urlPairs[i].ShortURL),
		}
	}
	resp := &pb.ShortenURLsResponse{URLs: shortenedURLs}

	return resp, nil
}

// GetUserURLs возвращает список пар ссылок (исходная и сокращенная), добавленных пользователем.
func (s *Server) GetUserURLs(ctx context.Context, request *pb.GetUserURLsRequest) (*pb.GetUserURLsResponse, error) {
	const op = "get user urls"

	user, _ := customctx.GetUser(ctx)
	if user.IsNew {
		return nil, errors.Wrap(status.Error(codes.PermissionDenied, unauthorizedMessage), op)
	}

	urlPairs, err := s.store.GetUserURLs(ctx, user.ID)
	if err != nil {
		return nil, errors.Wrap(status.Error(codes.Internal, failedToGetURLsMessage), op)
	}

	urls := make([]*pb.UserURL, len(urlPairs))
	for i := 0; i < len(urlPairs); i++ {
		urls[i] = &pb.UserURL{
			OriginalURL:  urlPairs[i].OriginalURL,
			ShortenedURL: joinPath(s.baseURL, urlPairs[i].ShortURL),
		}
	}
	resp := &pb.GetUserURLsResponse{
		URLs: urls,
	}

	return resp, nil
}

// GetDeleteUserURLs удаляет сокращенные ссылки по ключам.
//
//nolint:lll // naturally long name
func (s *Server) GetDeleteUserURLs(ctx context.Context, request *pb.DeleteUserURLsRequest) (*pb.DeleteUserURLsResponse, error) {
	const op = "delete user urls"

	user, _ := customctx.GetUser(ctx)

	var err error
	if s.urlRemover != nil {
		err = s.urlRemover.DeleteURLs(request.Keys, user.ID)
	} else {
		err = s.store.DeleteUserURLs(ctx, request.Keys, user.ID)
	}
	if err != nil {
		return nil, errors.Wrap(status.Error(codes.Internal, failedToDeleteURLsMessage), op)
	}

	return &pb.DeleteUserURLsResponse{}, nil
}

func joinPath(base, elem string) string {
	return fmt.Sprintf("%s/%s", base, elem)
}

func isTooManyURLs(urls []*pb.CorrelatedURL, maxCount int) bool {
	return maxCount > 0 && len(urls) > maxCount
}
