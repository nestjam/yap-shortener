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

func main() {
	config := conf.New().
		FromArgs(os.Args).
		FromEnv(env.New())

	logger := setupLogger()
	defer tearDown(logger)

	store := store.NewInMemory()
	server := server.New(store, config.BaseURL)

	logger.Info("Running server", zap.String("address", config.ServerAddress))
	if err := http.ListenAndServe(config.ServerAddress, server); err != nil {
		logger.Fatal(err.Error(), zap.String("event", "start server"))
	}
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
