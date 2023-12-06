package store

import (
	"context"

	"github.com/jackc/pgx/v5"
)

const notImplemented = "not implemented"

type SQLStorage struct {
	connString string
}

func NewSQLStorage(connString string) *SQLStorage {
	return &SQLStorage{
		connString,
	}
}

func (s *SQLStorage) Get(shortURL string) (string, error) {
	panic(notImplemented)
}

func (s *SQLStorage) Add(shortURL, url string) {
	panic(notImplemented)
}

func (s *SQLStorage) IsAvailable() bool {
	conn, err := pgx.Connect(context.Background(), s.connString)

	if err != nil {
		return false
	}

	defer func() {
		_ = conn.Close(context.Background())
	}()

	return true
}
