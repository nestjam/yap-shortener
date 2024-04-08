package pgsql

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"

	"github.com/nestjam/yap-shortener/internal/domain"
	"github.com/nestjam/yap-shortener/migration"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
)

// PostgresURLStore реализует хранилище ссылок в БД.
type PostgresURLStore struct {
	pool       *pgxpool.Pool
	connString string
}

// New создает экземпляр хранилища.
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

// Close завершает работу с БД.
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

// GetOriginalURL возвращает исходный URL для сокращенного URL или ошибку.
func (u *PostgresURLStore) GetOriginalURL(ctx context.Context, shortURL string) (string, error) {
	const op = "get original URL"
	conn, err := u.pool.Acquire(ctx)
	defer conn.Release()

	if err != nil {
		return "", errors.Wrapf(err, op)
	}

	var originalURL string
	var isDeleted bool
	row := conn.QueryRow(ctx, "SELECT original_url, is_deleted FROM url WHERE short_url=$1", shortURL)
	err = row.Scan(&originalURL, &isDeleted)

	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrOriginalURLNotFound
	}

	if isDeleted {
		return "", domain.ErrOriginalURLIsDeleted
	}

	return originalURL, nil
}

// AddURL добавляет в хранилище пару исходный и сокращенный URL.
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
		shortURL, er := getShortURL(ctx, conn, pair.OriginalURL)

		if er != nil {
			return errors.Wrapf(er, op)
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

// AddURLs добавляет в хранилище коллекцию пар исходного и сокращенного URL.
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

// IsAvailable позволяет проверить доступность хранилща.
func (u *PostgresURLStore) IsAvailable(ctx context.Context) bool {
	conn, err := u.pool.Acquire(ctx)
	defer conn.Release()

	if err != nil {
		return false
	}

	err = conn.Ping(ctx)
	return err == nil
}

// GetUserURLs возвращает коллекцию пар исходного и сокращенного URL, которые были добавлены указанным пользователем.
func (u *PostgresURLStore) GetUserURLs(ctx context.Context, userID domain.UserID) ([]domain.URLPair, error) {
	const op = "get user URLs"
	conn, err := u.pool.Acquire(ctx)
	defer conn.Release()

	if err != nil {
		return nil, errors.Wrapf(err, op)
	}

	const sql = "SELECT short_url, original_url FROM url WHERE user_id = $1 AND is_deleted = false"
	rows, err := conn.Query(ctx, sql, uuid.UUID(userID))

	if err != nil {
		return nil, errors.Wrapf(err, op)
	}

	defer rows.Close()

	var userURLs []domain.URLPair
	for rows.Next() {
		userURL := domain.URLPair{}
		err = rows.Scan(&userURL.ShortURL, &userURL.OriginalURL)

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

// DeleteUserURLs удаляет из хранилища коллекцию пар исходного и сокращенного URL,
// которые были добавлены указанным пользователем.
func (u *PostgresURLStore) DeleteUserURLs(ctx context.Context, shortURLs []string, userID domain.UserID) error {
	const op = "delete user URLs"
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

	b := &pgx.Batch{}

	for i := 0; i < len(shortURLs); i++ {
		b.Queue("UPDATE url SET is_deleted = true WHERE short_url = $1 AND user_id = $2", shortURLs[i], uuid.UUID(userID))
	}

	err = tx.SendBatch(ctx, b).Close()

	if err != nil {
		return errors.Wrapf(err, op)
	}

	err = tx.Commit(ctx)

	if err != nil {
		return errors.Wrapf(err, op)
	}

	return nil
}

// GetURLsAndUsersCount возвращает количество сокращенных ссылок и пользователей.
func (u *PostgresURLStore) GetURLsAndUsersCount(ctx context.Context) (urlsCount, usersCount int, err error) {
	const op = "get URLs and users count"
	conn, err := u.pool.Acquire(ctx)
	defer conn.Release()

	if err != nil {
		err = errors.Wrapf(err, op)
		return
	}

	row := conn.QueryRow(ctx, "SELECT COUNT(short_url) AS urls_count, COUNT(DISTINCT user_id) AS users_count FROM url")
	err = row.Scan(&urlsCount, &usersCount)

	if err != nil {
		err = errors.Wrapf(err, op)
		return
	}

	return
}
