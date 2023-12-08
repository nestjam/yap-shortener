package pgsql

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nestjam/yap-shortener/internal/domain"
	"github.com/pkg/errors"
)

type URLStore struct {
	pool       *pgxpool.Pool
	connString string
}

func New(connString string) *URLStore {
	return &URLStore{
		connString: connString,
	}
}

func (u *URLStore) Init() error {
	const op = "init store"
	var err error
	u.pool, err = initPool(context.Background(), u.connString)

	if err != nil {
		return errors.Wrapf(err, op)
	}

	conn, err := u.pool.Acquire(context.Background())
	defer conn.Release()

	if err != nil {
		return errors.Wrapf(err, op)
	}

	_, err = conn.Exec(context.Background(), `CREATE TABLE IF NOT EXISTS url(id SERIAL PRIMARY KEY,
		short_url VARCHAR(255),
		original_url TEXT);`)

	if err != nil {
		return errors.Wrapf(err, op)
	}

	return nil
}

func (u *URLStore) Close() {
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

func (u *URLStore) Get(shortURL string) (string, error) {
	const op = "get original url"
	conn, err := u.pool.Acquire(context.Background())
	defer conn.Release()

	if err != nil {
		return "", errors.Wrapf(err, op)
	}

	var originalURL string
	row := conn.QueryRow(context.Background(), "SELECT original_url FROM url WHERE short_url=$1", shortURL)
	err = row.Scan(&originalURL)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrURLNotFound
	}

	return originalURL, nil
}

func (u *URLStore) Add(shortURL, url string) error {
	const op = "add url"
	conn, err := u.pool.Acquire(context.Background())
	defer conn.Release()

	if err != nil {
		return errors.Wrapf(err, op)
	}

	_, err = conn.Exec(context.Background(), "INSERT INTO url (short_url, original_url) VALUES ($1, $2)",
		shortURL, url)

	if err != nil {
		return errors.Wrapf(err, op)
	}

	return nil
}

func (u *URLStore) AddBatch(pairs []domain.URLPair) error {
	const op = "add batch of urls"
	conn, err := u.pool.Acquire(context.Background())
	defer conn.Release()

	if err != nil {
		return errors.Wrapf(err, op)
	}

	var txOptions pgx.TxOptions
	tx, err := conn.BeginTx(context.Background(), txOptions)

	if err != nil {
		return errors.Wrapf(err, op)
	}

	defer func() {
		_ = tx.Rollback(context.Background())
	}()

	stmt, params := prepareInsert(pairs)
	_, err = tx.Exec(context.Background(), stmt, params...)

	if err != nil {
		return errors.Wrapf(err, op)
	}

	err = tx.Commit(context.Background())

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

func (u *URLStore) IsAvailable() bool {
	conn, err := u.pool.Acquire(context.Background())
	defer conn.Release()

	if err != nil {
		return false
	}

	err = conn.Ping(context.Background())
	return err == nil
}
