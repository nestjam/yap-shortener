package migration

import (
	"embed"

	"github.com/pkg/errors"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed *.sql
var migrationsDir embed.FS

// URLStoreMigrator выполняет миграцию данных.
type URLStoreMigrator struct {
	connString string
}

// NewURLStoreMigrator создает экземпляр URLStoreMigrator.
func NewURLStoreMigrator(connString string) *URLStoreMigrator {
	return &URLStoreMigrator{connString: connString}
}

// Up выполняет обновление данных до последней версии.
func (u *URLStoreMigrator) Up() error {
	const op = "migrate up"
	m, err := createMigrate(u.connString)

	if err != nil {
		return errors.Wrapf(err, op)
	}

	if err := m.Up(); err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			return errors.Wrapf(err, op)
		}
	}

	return nil
}

func createMigrate(connString string) (*migrate.Migrate, error) {
	const (
		op             = "create migrate"
		migrationsPath = "."
	)
	d, err := iofs.New(migrationsDir, migrationsPath)

	if err != nil {
		return nil, errors.Wrapf(err, op)
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, connString)

	if err != nil {
		return nil, errors.Wrapf(err, op)
	}

	return m, nil
}

// Drop удаляет все в БД.
func (u *URLStoreMigrator) Drop() error {
	const op = "drop"
	m, err := createMigrate(u.connString)

	if err != nil {
		return errors.Wrapf(err, op)
	}

	err = m.Drop()

	if err != nil {
		return errors.Wrapf(err, op)
	}

	return nil
}
