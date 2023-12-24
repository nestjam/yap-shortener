package context

import (
	"context"

	"github.com/nestjam/yap-shortener/internal/domain"
)

type contextKey int

const (
	userIDContextKey contextKey = iota
)

type User struct {
	ID    domain.UserID
	IsNew bool
}

func NewUser(id domain.UserID, isNew bool) User {
	return User{
		ID:    id,
		IsNew: isNew,
	}
}

func SetUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, userIDContextKey, user)
}

func GetUser(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(userIDContextKey).(User)
	return user, ok
}
