package main

import (
	"context"
	"net/http"
	"os"

	"go.uber.org/zap"
	"golang.org/x/crypto/acme/autocert"

	conf "github.com/nestjam/yap-shortener/internal/config"
	env "github.com/nestjam/yap-shortener/internal/config/environment"
	factory "github.com/nestjam/yap-shortener/internal/factory"
	"github.com/nestjam/yap-shortener/internal/server"
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
	store, tearDownStorage := factory.NewStorage(ctx, config, logger)
	defer tearDownStorage()

	doneCh := make(chan struct{})
	defer close(doneCh)

	urlRemoved := server.NewURLRemover(ctx, doneCh, store, logger)

	handler := server.New(store, config.BaseURL,
		server.WithLogger(logger),
		server.WithShortenURLsMaxCount(shortenURLsMaxCount),
		server.WithURLsRemover(urlRemoved))

	runServer(config, handler, logger)
}

func runServer(config conf.Config, handler *server.Server, logger *zap.Logger) {
	server := &http.Server{
		Addr:    config.ServerAddress,
		Handler: handler,
	}

	logger.Info("Running server", zap.String("address", config.ServerAddress))
	var err error

	if config.EnableHTTPS {
		manager := &autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist("shrt.ru"),
		}
		server.TLSConfig = manager.TLSConfig()
		const (
			certFile = ""
			keyfile  = ""
		)

		err = server.ListenAndServeTLS(certFile, keyfile)
	} else {
		err = server.ListenAndServe()
	}

	if err != nil {
		logger.Fatal(err.Error(), zap.String(eventKey, "start server"))
	}
}
