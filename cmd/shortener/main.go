package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"go.uber.org/zap"

	conf "github.com/nestjam/yap-shortener/internal/config"
	env "github.com/nestjam/yap-shortener/internal/config/environment"
	factory "github.com/nestjam/yap-shortener/internal/factory"
	"github.com/nestjam/yap-shortener/internal/server"
)

const (
	eventKey            = "event"
	shortenURLsMaxCount = 1000
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	printBuildInfo()

	config := conf.New().
		FromArgs(os.Args).
		FromEnv(env.New())

	logger, tearDownLogger := factory.NewLogger()
	defer tearDownLogger()

	ctx := context.Background()
	store, tearDownStorage := factory.NewStorage(ctx, config, logger)
	defer tearDownStorage()

	doneCh := make(chan struct{})
	defer close(doneCh)

	urlRemoved := server.NewURLRemover(ctx, doneCh, store, logger)

	server := server.New(store, config.BaseURL,
		server.WithLogger(logger),
		server.WithShortenURLsMaxCount(shortenURLsMaxCount),
		server.WithURLsRemover(urlRemoved))
	listenAndServe(config.ServerAddress, server, logger)
}

func printBuildInfo() {
	const notAwailable = "N/A"

	if buildVersion == "" {
		fmt.Println(notAwailable)
	} else {
		fmt.Printf("Build version: %s\n", buildVersion)
	}

	if buildDate == "" {
		fmt.Println(notAwailable)
	} else {
		fmt.Printf("Build date: %s\n", buildDate)
	}

	if buildCommit == "" {
		fmt.Println(notAwailable)
	} else {
		fmt.Printf("Build commit: %s\n", buildCommit)
	}
}

func listenAndServe(address string, server *server.Server, logger *zap.Logger) {
	logger.Info("Running server", zap.String("address", address))
	if err := http.ListenAndServe(address, server); err != nil {
		logger.Fatal(err.Error(), zap.String(eventKey, "start server"))
	}
}
