package domain

import (
	"errors"
	"fmt"
)

var (
	ErrOriginalURLNotFound  = errors.New("not found")
	ErrOriginalURLIsDeleted = errors.New("url is deleted")
)

type OriginalURLExistsError struct {
	err      error
	shortURL string
}

func NewOriginalURLExistsError(shortURL string, err error) *OriginalURLExistsError {
	return &OriginalURLExistsError{
		err:      err,
		shortURL: shortURL,
	}
}

func (u *OriginalURLExistsError) Error() string {
	return fmt.Sprintf("original URL already exists: %v", u.err.Error())
}

func (u *OriginalURLExistsError) GetShortURL() string {
	return u.shortURL
}
