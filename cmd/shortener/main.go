package main

import (
	"context"
	"net/http"
	"os"

	conf "github.com/nestjam/yap-shortener/internal/config"
	env "github.com/nestjam/yap-shortener/internal/config/environment"
	factory "github.com/nestjam/yap-shortener/internal/factory"
	"github.com/nestjam/yap-shortener/internal/server"
	"go.uber.org/zap"
)

const (
	eventKey            = "event"
	shortenURLsMaxCount = 1000
)

func main() {
	config := conf.New().
		FromArgs(os.Args).
		FromEnv(env.New())

	logger, tearDownLogger := factory.NewLogger()
	defer tearDownLogger()

	ctx := context.Background()
	storage, tearDownStorage := factory.NewStorage(ctx, config, logger)
	defer tearDownStorage()

	server := server.New(storage, config.BaseURL, logger, server.WithShortenURLsMaxCount(shortenURLsMaxCount))
	listenAndServe(config.ServerAddress, server, logger)
}

func listenAndServe(address string, server *server.Server, logger *zap.Logger) {
	logger.Info("Running server", zap.String("address", address))
	if err := http.ListenAndServe(address, server); err != nil {
		logger.Fatal(err.Error(), zap.String(eventKey, "start server"))
	}
}
