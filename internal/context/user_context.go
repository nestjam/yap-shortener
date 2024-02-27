package context

import (
	"context"

	"github.com/nestjam/yap-shortener/internal/domain"
)

type contextKey int

const (
	userIDContextKey contextKey = iota
)

// User содержит информацию о пользователе.
type User struct {
	ID    domain.UserID // идентификатор пользователя
	IsNew bool          // признак нового пользователя
}

// NewUser создает экземпляр пользователя.
func NewUser(id domain.UserID, isNew bool) User {
	return User{
		ID:    id,
		IsNew: isNew,
	}
}

// SetUser возвращает контекст с добавленным пользователем.
func SetUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, userIDContextKey, user)
}

// GetUser получает пользователя из контекста, если пользователь добавлен в контекст.
func GetUser(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(userIDContextKey).(User)
	return user, ok
}
