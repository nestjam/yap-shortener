package domain

import (
	"errors"
	"fmt"
)

// Ошибки, связанные с исходным URL.
var (
	ErrOriginalURLNotFound  = errors.New("not found") // исходный URL не найден
	ErrOriginalURLIsDeleted = errors.New("url is deleted") // исходный URL удален
)

// OriginalURLExistsError определяет ошибку, когда исходный URL уже был сокращен.
type OriginalURLExistsError struct {
	err      error
	shortURL string
}

// NewOriginalURLExistsError создает экземпляр ошибки.
func NewOriginalURLExistsError(shortURL string, err error) *OriginalURLExistsError {
	return &OriginalURLExistsError{
		err:      err,
		shortURL: shortURL,
	}
}

// Error возвращает текст ошибки.
func (u *OriginalURLExistsError) Error() string {
	return fmt.Sprintf("original URL already exists: %v", u.err.Error())
}

// GetShortUR возвращает сокращенный URL, который был создан ранее.
func (u *OriginalURLExistsError) GetShortURL() string {
	return u.shortURL
}
