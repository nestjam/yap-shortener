package pgsql

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/nestjam/yap-shortener/internal/domain"
	"github.com/stretchr/testify/require"
)

const connString = "postgres://postgres:postgres@localhost:5432/praktikum"

func TestURLStore(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running test.")
	}
	domain.URLStoreContract{
		NewURLStore: func() (domain.URLStore, func()) {
			t.Helper()

			if !pingDB(t, connString) {
				t.Skip("Skipping test: unavailable database.")
			}

			store := New(connString)
			err := store.Init()

			require.NoError(t, err)

			return store, func() {
				dropTable(t)
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
