package grpc

import (
	"context"

	"github.com/google/uuid"
	"github.com/nestjam/yap-shortener/internal/domain"
	"github.com/nestjam/yap-shortener/internal/shortener"
	pb "github.com/nestjam/yap-shortener/proto"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	customctx "github.com/nestjam/yap-shortener/internal/context"
)

const (
	urlIsEmptyMessage = "url is empty"
)

// Server предоставляет возможность сокращать URL, получать исходный и управлять сокращенными URL.
type Server struct {
	pb.UnimplementedShortenerServer
	store   domain.URLStore
	baseURL string
}

// Option определяет опцию настройки сервера.
type Option func(*Server)

// New создает сервер. Конструктор принимает на вход хранилище URL, базовый URL и набор опций.
func New(store domain.URLStore, baseURL string, options ...Option) *Server {
	s := &Server{
		store:   store,
		baseURL: baseURL,
	}

	for _, opt := range options {
		opt(s)
	}

	return s
}

// Ping проверяет доступность сервиса.
func (s *Server) Ping(ctx context.Context, request *pb.PingRequest) (*pb.PingResponse, error) {
	response := &pb.PingResponse{
		Result: s.store.IsAvailable(ctx),
	}
	return response, nil
}

// GetOriginalURL возвращает исходный URL по ключу.
//nolint: lll
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
		return nil, errors.Wrap(status.Error(codes.Internal, err.Error()), op)
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
