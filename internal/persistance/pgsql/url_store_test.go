package pgsql

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/nestjam/yap-shortener/internal/domain"
	"github.com/stretchr/testify/require"
)

const connString = "postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable"

func TestURLStore(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running test.")
	}
	if !pingDB(t, connString) {
		t.Skip("Skipping test: unavailable database.")
	}
	domain.URLStoreContract{
		NewURLStore: func() (domain.URLStore, func()) {
			t.Helper()
			store := New(connString)
			err := store.Init()

			require.NoError(t, err)

			return store, func() {
				store.Close()

				migrator := NewURLStoreMigrator(connString)
				_ = migrator.Drop()
			}
		},
	}.Test(t)
}

func pingDB(t *testing.T, connString string) bool {
	t.Helper()

	conn, err := pgx.Connect(context.Background(), connString)

	if err != nil {
		return false
	}

	defer func() {
		_ = conn.Close(context.Background())
	}()

	return conn.Ping(context.Background()) == nil
}
