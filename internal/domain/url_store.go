package domain

import "errors"

var (
	ErrNotFound = errors.New("not found")
)

type URLStore interface {
	Get(shortURL string) (string, error)

	Add(shortURL, url string)

	IsAvailable() bool
}
