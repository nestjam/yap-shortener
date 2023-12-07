package pgsql

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/nestjam/yap-shortener/internal/domain"
	"github.com/pkg/errors"
)

type URLStore struct {
	connString string
}

func New(connString string) *URLStore {
	return &URLStore{
		connString,
	}
}

func (s *URLStore) Init() error {
	const op = "init store"
	conn, err := pgx.Connect(context.Background(), s.connString)

	if err != nil {
		return errors.Wrapf(err, op)
	}

	defer func() {
		_ = conn.Close(context.Background())
	}()

	_, err = conn.Exec(context.Background(), `CREATE TABLE IF NOT EXISTS url(id SERIAL PRIMARY KEY,
		short_url VARCHAR(255),
		original_url TEXT);`)

	if err != nil {
		return errors.Wrapf(err, op)
	}

	return nil
}

func (s *URLStore) Get(shortURL string) (string, error) {
	const op = "get original url"
	conn, err := pgx.Connect(context.Background(), s.connString)

	if err != nil {
		return "", errors.Wrapf(err, op)
	}

	defer func() {
		_ = conn.Close(context.Background())
	}()

	var originalURL string
	row := conn.QueryRow(context.Background(), "SELECT original_url FROM url WHERE short_url=$1", shortURL)
	err = row.Scan(&originalURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrURLNotFound
	}

	return originalURL, nil
}

func (s *URLStore) Add(shortURL, url string) {
	conn, _ := pgx.Connect(context.Background(), s.connString)

	defer func() {
		_ = conn.Close(context.Background())
	}()

	_ = conn.QueryRow(context.Background(), "INSERT INTO url (short_url, original_url) VALUES ($1, $2)",
		shortURL, url)
}

func (s *URLStore) IsAvailable() bool {
	conn, err := pgx.Connect(context.Background(), s.connString)

	if err != nil {
		return false
	}

	defer func() {
		_ = conn.Close(context.Background())
	}()

	err = conn.Ping(context.Background())
	return err == nil
}
