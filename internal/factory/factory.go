package factory

import (
	"os"

	conf "github.com/nestjam/yap-shortener/internal/config"
	"github.com/nestjam/yap-shortener/internal/log"
	"github.com/nestjam/yap-shortener/internal/server"
	"github.com/nestjam/yap-shortener/internal/store"
	"go.uber.org/zap"
)

const (
	eventKey = "event"
)

func NewStorage(conf conf.Config, logger *zap.Logger) (server.URLStorage, func()) {
	if conf.FileStoragePath == "" {
		logger.Info("Using in-memory storage")
		return store.NewInMemory(), func() {}
	}

	logger.Info("Using file storage", zap.String("path", conf.FileStoragePath))
	return newFileStorage(conf, logger)
}

func newFileStorage(conf conf.Config, logger *zap.Logger) (server.URLStorage, func()) {
	const ownerReadWritePermission os.FileMode = 0600
	file, err := os.OpenFile(conf.FileStoragePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, ownerReadWritePermission)
	if err != nil {
		logger.Fatal(err.Error(), zap.String(eventKey, "open file"))
	}

	store, err := store.NewFileStorage(file)
	if err != nil {
		logger.Fatal(err.Error(), zap.String(eventKey, "create storage"))
	}

	return store, func() { _ = file.Close() }
}

func NewLogger() (*zap.Logger, func()) {
	logger, err := log.NewProductionLogger()

	if err != nil {
		panic(err)
	}

	return logger, func() { _ = logger.Sync() }
}
