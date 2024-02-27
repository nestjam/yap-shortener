package factory

import (
	"context"
	"fmt"
	"os"

	conf "github.com/nestjam/yap-shortener/internal/config"
	"github.com/nestjam/yap-shortener/internal/domain"
	filestore "github.com/nestjam/yap-shortener/internal/persistance/file"
	"github.com/nestjam/yap-shortener/internal/persistance/inmemory"
	"github.com/nestjam/yap-shortener/internal/persistance/pgsql"
	"go.uber.org/zap"
)

const (
	eventKey = "event"
)

// NewStorage создает экземпляр хранилища на основе конфигурации.
// Возвращает хранилище и функцию для корректного закрытия хранилища.
func NewStorage(ctx context.Context, conf conf.Config, logger *zap.Logger) (domain.URLStore, func()) {
	if conf.DataSourceName != "" {
		logger.Info("Using sql store")
		return newPGSQLStore(ctx, conf, logger)
	}

	if conf.FileStoragePath != "" {
		logger.Info("Using file store", zap.String("path", conf.FileStoragePath))
		return newFileStore(ctx, conf, logger)
	}

	logger.Info("Using in-memory store")
	closer := func() {}
	return inmemory.New(), closer
}

func newPGSQLStore(ctx context.Context, conf conf.Config, logger *zap.Logger) (domain.URLStore, func()) {
	store, err := pgsql.New(ctx, conf.DataSourceName)

	if err != nil {
		logger.Fatal("Failed to initialize store", zap.Error(err))
	}

	closer := func() { store.Close() }
	return store, closer
}

func newFileStore(ctx context.Context, conf conf.Config, logger *zap.Logger) (domain.URLStore, func()) {
	const ownerReadWritePermission os.FileMode = 0600
	file, err := os.OpenFile(conf.FileStoragePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, ownerReadWritePermission)
	if err != nil {
		logger.Fatal(err.Error(), zap.String(eventKey, "open file"))
	}

	store, err := filestore.New(ctx, file)
	if err != nil {
		logger.Fatal(err.Error(), zap.String(eventKey, "create store"))
	}

	closer := func() { _ = file.Close() }
	return store, closer
}

// NewLogger создает экземпляр логгера.
// Возвращает логгер и функцию для корректного закрытия логгера.
func NewLogger() (*zap.Logger, func()) {
	logger, err := newProductionLogger()

	if err != nil {
		panic(err)
	}

	closer := func() { _ = logger.Sync() }
	return logger, closer
}

func newProductionLogger() (*zap.Logger, error) {
	const op = "new production logger"
	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	logger, err := config.Build()

	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return logger, nil
}
