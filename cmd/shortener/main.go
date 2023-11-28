package main

import (
	"net/http"
	"os"

	conf "github.com/nestjam/yap-shortener/internal/config"
	env "github.com/nestjam/yap-shortener/internal/config/environment"
	"github.com/nestjam/yap-shortener/internal/log"
	"github.com/nestjam/yap-shortener/internal/server"
	"github.com/nestjam/yap-shortener/internal/store"
	"go.uber.org/zap"
)

const (
	eventKey = "event"
)

func main() {
	config := conf.New().
		FromArgs(os.Args).
		FromEnv(env.New())

	logger := setupLogger()
	defer tearDown(logger)

	storage, tearDownStorage := newStorage(config, logger)
	defer tearDownStorage()

	server := server.New(storage, config.BaseURL)
	listenAndServe(config.ServerAddress, server, logger)
}

func listenAndServe(address string, server *server.Server, logger *zap.Logger) {
	logger.Info("Running server", zap.String("address", address))
	if err := http.ListenAndServe(address, server); err != nil {
		logger.Fatal(err.Error(), zap.String(eventKey, "start server"))
	}
}

func newStorage(conf conf.Config, logger *zap.Logger) (server.URLStorage, func()) {
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

func tearDown(logger *zap.Logger) {
	_ = logger.Sync()
}

func setupLogger() *zap.Logger {
	if err := log.Initialize(); err != nil {
		panic(err)
	}
	return log.Logger
}
