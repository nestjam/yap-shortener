package domain

import (
	"github.com/google/uuid"
)

// UserID определяет идентификатор пользователя.
type UserID uuid.UUID

// NewUserID возвращает новый идентифиатор пользователя.
func NewUserID() UserID {
	return UserID(uuid.New())
}
