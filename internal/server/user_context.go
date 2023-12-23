package server

import (
	"context"

	"github.com/nestjam/yap-shortener/internal/domain"
)

type contextKey int

const (
	userIDContextKey contextKey = iota
)

func SetUserID(ctx context.Context, userID domain.UserID) context.Context {
	return context.WithValue(ctx, userIDContextKey, userID)
}

func GetUserID(ctx context.Context) (domain.UserID, bool) {
	userID, ok := ctx.Value(userIDContextKey).(domain.UserID)
	return userID, ok
}
