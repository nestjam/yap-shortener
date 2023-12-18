package pgsql

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nestjam/yap-shortener/internal/domain"
	"github.com/pkg/errors"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
)

type PostgresURLStore struct {
	pool       *pgxpool.Pool
	connString string
}

func New(connString string) *PostgresURLStore {
	return &PostgresURLStore{
		connString: connString,
	}
}

func (u *PostgresURLStore) Init(ctx context.Context) error {
	const op = "init store"

	migrator := NewURLStoreMigrator(u.connString)
	if err := migrator.Up(); err != nil {
		return errors.Wrapf(err, op)
	}

	var err error
	u.pool, err = initPool(ctx, u.connString)

	if err != nil {
		return errors.Wrapf(err, op)
	}

	return nil
}

func (u *PostgresURLStore) Close() {
	if u.pool == nil {
		return
	}
	u.pool.Close()
}

func initPool(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	const op = "init connection pool"
	poolCfg, err := pgxpool.ParseConfig(connString)

	if err != nil {
		return nil, errors.Wrapf(err, op)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)

	if err != nil {
		return nil, errors.Wrapf(err, op)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, errors.Wrapf(err, op)
	}

	return pool, nil
}

func (u *PostgresURLStore) GetOriginalURL(ctx context.Context, shortURL string) (string, error) {
	const op = "get original URL"
	conn, err := u.pool.Acquire(ctx)
	defer conn.Release()

	if err != nil {
		return "", errors.Wrapf(err, op)
	}

	var originalURL string
	row := conn.QueryRow(ctx, "SELECT original_url FROM url WHERE short_url=$1", shortURL)
	err = row.Scan(&originalURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrOriginalURLNotFound
	}

	return originalURL, nil
}

func (u *PostgresURLStore) AddURL(ctx context.Context, shortURL, originalURL string) error {
	const op = "add URL"

	conn, err := u.pool.Acquire(ctx)
	defer conn.Release()

	if err != nil {
		return errors.Wrapf(err, op)
	}

	var txOptions pgx.TxOptions
	tx, err := conn.BeginTx(ctx, txOptions)

	if err != nil {
		return errors.Wrapf(err, op)
	}

	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, "INSERT INTO url (short_url, original_url) VALUES ($1, $2)",
		shortURL, originalURL)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
		_ = tx.Rollback(ctx)
		shortURL, err := getShortURL(ctx, conn, originalURL)

		if err != nil {
			return errors.Wrapf(err, op)
		}

		return domain.NewOriginalURLExistsError(shortURL, nil)
	}

	if err != nil {
		return errors.Wrapf(err, op)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return errors.Wrapf(err, op)
	}

	return nil
}

func getShortURL(ctx context.Context, conn *pgxpool.Conn, originalURL string) (string, error) {
	var shortURL string
	row := conn.QueryRow(ctx, "SELECT short_url FROM url WHERE original_url=$1", originalURL)
	err := row.Scan(&shortURL)

	if err != nil {
		return "", fmt.Errorf("get short url: %w", err)
	}

	return shortURL, nil
}

func (u *PostgresURLStore) AddURLs(ctx context.Context, pairs []domain.URLPair) error {
	const op = "add URLs"
	conn, err := u.pool.Acquire(ctx)
	defer conn.Release()

	if err != nil {
		return errors.Wrapf(err, op)
	}

	var txOptions pgx.TxOptions
	tx, err := conn.BeginTx(ctx, txOptions)

	if err != nil {
		return errors.Wrapf(err, op)
	}

	defer func() { _ = tx.Rollback(ctx) }()

	stmt, params := prepareInsert(pairs)
	_, err = tx.Exec(ctx, stmt, params...)

	if err != nil {
		return errors.Wrapf(err, op)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return errors.Wrapf(err, op)
	}

	return nil
}

func prepareInsert(pairs []domain.URLPair) (string, []any) {
	const paramsInPair = 2
	b := strings.Builder{}
	b.WriteString("INSERT INTO url (short_url, original_url) VALUES ")

	params := make([]any, len(pairs)*paramsInPair)
	for i := 0; i < len(pairs); i++ {
		first := i * paramsInPair
		second := first + 1
		params[first] = pairs[i].ShortURL
		params[second] = pairs[i].OriginalURL
		_, _ = b.WriteString(fmt.Sprintf("($%d, $%d),", first+1, second+1))
	}

	return strings.TrimSuffix(b.String(), ","), params
}

func (u *PostgresURLStore) IsAvailable(ctx context.Context) bool {
	conn, err := u.pool.Acquire(ctx)
	defer conn.Release()

	if err != nil {
		return false
	}

	err = conn.Ping(ctx)
	return err == nil
}
