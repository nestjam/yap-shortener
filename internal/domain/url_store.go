package domain

import "errors"

var (
	ErrURLNotFound = errors.New("not found")
)

type URLStore interface {
	Get(shortURL string) (string, error)

	Add(shortURL, url string)

	IsAvailable() bool
}
