package pgsql

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nestjam/yap-shortener/internal/domain"
	"github.com/nestjam/yap-shortener/migration"
	"github.com/pkg/errors"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
)

type PostgresURLStore struct {
	pool       *pgxpool.Pool
	connString string
}

func New(ctx context.Context, connString string) (*PostgresURLStore, error) {
	const op = "new store"

	migrator := migration.NewURLStoreMigrator(connString)
	if err := migrator.Up(); err != nil {
		return nil, errors.Wrapf(err, op)
	}

	var err error
	pool, err := initPool(ctx, connString)

	if err != nil {
		return nil, errors.Wrapf(err, op)
	}

	store := &PostgresURLStore{
		pool,
		connString,
	}
	return store, nil
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

func (u *PostgresURLStore) AddURL(ctx context.Context, pair domain.URLPair, userID domain.UserID) error {
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

	_, err = tx.Exec(ctx, "INSERT INTO url (short_url, original_url, user_id) VALUES ($1, $2, $3)",
		pair.ShortURL, pair.OriginalURL, uuid.UUID(userID))

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
		_ = tx.Rollback(ctx)
		shortURL, err := getShortURL(ctx, conn, pair.OriginalURL)

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

func (u *PostgresURLStore) AddURLs(ctx context.Context, pairs []domain.URLPair, userID domain.UserID) error {
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

	columns := []string{"short_url", "original_url", "user_id"}
	rows := pgx.CopyFromRows(prepareRows(pairs, userID))
	_, err = conn.CopyFrom(ctx, pgx.Identifier{"url"}, columns, rows)

	if err != nil {
		return errors.Wrapf(err, op)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return errors.Wrapf(err, op)
	}

	return nil
}

func prepareRows(pairs []domain.URLPair, userID domain.UserID) [][]any {
	rows := make([][]any, len(pairs))

	for i := 0; i < len(pairs); i++ {
		pair := pairs[i]
		rows[i] = []any{pair.ShortURL, pair.OriginalURL, uuid.UUID(userID)}
	}

	return rows
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

func (u *PostgresURLStore) GetUserURLs(ctx context.Context, userID domain.UserID) ([]domain.URLPair, error) {
	const op = "get user URLs"
	conn, err := u.pool.Acquire(ctx)
	defer conn.Release()

	if err != nil {
		return nil, errors.Wrapf(err, op)
	}

	rows, err := conn.Query(ctx, "SELECT short_url, original_url FROM url WHERE user_id = $1", uuid.UUID(userID))

	if err != nil {
		return nil, errors.Wrapf(err, op)
	}

	defer rows.Close()

	var userURLs []domain.URLPair
	for rows.Next() {
		userURL := domain.URLPair{}
		err := rows.Scan(&userURL.ShortURL, &userURL.OriginalURL)

		if err != nil {
			return nil, errors.Wrapf(err, op)
		}

		userURLs = append(userURLs, userURL)
	}

	if rows.Err() != nil {
		return nil, errors.Wrapf(err, op)
	}

	return userURLs, nil
}
