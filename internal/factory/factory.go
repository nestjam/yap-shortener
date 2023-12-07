package factory

import (
	"fmt"
	"os"

	conf "github.com/nestjam/yap-shortener/internal/config"
	"github.com/nestjam/yap-shortener/internal/server"
	f "github.com/nestjam/yap-shortener/internal/store/file"
	"github.com/nestjam/yap-shortener/internal/store/inmemory"
	"github.com/nestjam/yap-shortener/internal/store/pgsql"
	"go.uber.org/zap"
)

const (
	eventKey = "event"
)

func NewStorage(conf conf.Config, logger *zap.Logger) (server.URLStorage, func()) {
	if conf.DataSourceName != "" {
		logger.Info("Using sql storage")
		store := pgsql.New(conf.DataSourceName)
		err := store.Init()

		if err != nil {
			logger.Fatal("Failed to initialize store", zap.Error(err))
		}

		return store, func() {}
	}

	if conf.FileStoragePath != "" {
		logger.Info("Using file storage", zap.String("path", conf.FileStoragePath))
		return newFileStorage(conf, logger)
	}

	logger.Info("Using in-memory storage")
	return inmemory.New(), func() {}
}

func newFileStorage(conf conf.Config, logger *zap.Logger) (server.URLStorage, func()) {
	const ownerReadWritePermission os.FileMode = 0600
	file, err := os.OpenFile(conf.FileStoragePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, ownerReadWritePermission)
	if err != nil {
		logger.Fatal(err.Error(), zap.String(eventKey, "open file"))
	}

	store, err := f.New(file)
	if err != nil {
		logger.Fatal(err.Error(), zap.String(eventKey, "create storage"))
	}

	return store, func() { _ = file.Close() }
}

func NewLogger() (*zap.Logger, func()) {
	logger, err := newProductionLogger()

	if err != nil {
		panic(err)
	}

	return logger, func() { _ = logger.Sync() }
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
