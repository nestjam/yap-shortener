package pgsql

import (
	"context"
	"testing"

	"github.com/nestjam/yap-shortener/internal/domain"
	"github.com/nestjam/yap-shortener/migration"
	"github.com/stretchr/testify/require"
)

const connString = "postgres://postgres:postgres@localhost:5432/praktikum?sslmode=disable"

func TestPostgresURLStore(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running test.")
	}
	domain.URLStoreContract{
		NewURLStore: func() (domain.URLStore, func()) {
			t.Helper()
			store, err := New(context.Background(), connString)

			require.NoError(t, err)

			return store, func() {
				store.Close()

				migrator := migration.NewURLStoreMigrator(connString)
				_ = migrator.Drop()
			}
		},
	}.Test(t)
}
