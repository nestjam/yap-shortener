package pgsql

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const connString = "postgres://postgres:postgres@localhost:5432/praktikum"

func TestAdd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running test.")
	}

	t.Run("add new url", func(t *testing.T) {
		const (
			shortURL    = "abc"
			originalURL = "http://example.com"
		)
		sut, tearDown := NewStorage(t)
		defer tearDown()

		sut.Add(shortURL, originalURL)

		got, err := sut.Get(shortURL)
		require.NoError(t, err)
		assert.Equal(t, originalURL, got)
	})
}

func TestGet(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running test.")
	}

	t.Run("original url not found by short url", func(t *testing.T) {
		sut, tearDown := NewStorage(t)
		defer tearDown()

		_, err := sut.Get("123")
		assert.Error(t, err)
	})
}

func TestIsAvailable(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running test.")
	}
	
	t.Run("store is available", func(t *testing.T) {
		sut, tearDown := NewStorage(t)
		defer tearDown()

		got := sut.IsAvailable()
		assert.True(t, got)
	})
}

func NewStorage(t *testing.T) (*SQLStorage, func()) {
	t.Helper()
	store := NewSQLStorage(connString)
	err := store.Init()

	require.NoError(t, err)

	return store, func() {
		dropTable(t)
	}
}

func dropTable(t *testing.T) {
	t.Helper()
	conn, err := pgx.Connect(context.Background(), connString)

	require.NoError(t, err)

	defer func() {
		_ = conn.Close(context.Background())
	}()

	_, err = conn.Exec(context.Background(), `DROP TABLE IF EXISTS url;`)

	require.NoError(t, err)
}
