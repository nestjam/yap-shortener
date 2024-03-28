package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	defer cancel()

	config := getConfig()

	logger, tearDownLogger := factory.NewLogger()
	defer tearDownLogger()

	store, tearDownStorage := factory.NewStorage(ctx, config, logger)
	defer tearDownStorage()

	doneCh := make(chan struct{})
	defer close(doneCh)
	urlRemoved := server.NewURLRemover(ctx, doneCh, store, logger)

	handler := server.New(store, config.BaseURL,
		server.WithLogger(logger),
		server.WithShortenURLsMaxCount(shortenURLsMaxCount),
		server.WithURLsRemover(urlRemoved))

	runServer(ctx, config, handler, logger)
}

func getConfig() conf.Config {
	config := conf.New()

	filePath := getConfigFilePath()
	if filePath != "" {
		data, err := os.ReadFile(filePath)

		if err != nil {
			panic(err)
		}

		config = config.FromJSON(data)
	}

	config = config.
		FromArgs(os.Args).
		FromEnv(env.New())

	return config
}

func getConfigFilePath() string {
	path := conf.GetConfigFileFromArgs(os.Args)

	pathFromEnv := conf.GetConfigFileFromEnv(env.New())
	if pathFromEnv != "" {
		path = pathFromEnv
	}

	return path
}

func runServer(ctx context.Context, config conf.Config, handler *server.Server, log *zap.Logger) {
	doneCh := make(chan struct{})

	server := &http.Server{
		Addr:    config.ServerAddress,
		Handler: handler,
	}

	go func() {
		<-ctx.Done()

		if err := server.Shutdown(context.Background()); err != nil {
			log.Sugar().Infof("HTTP server shut down: %v", err)
		}

		close(doneCh)
	}()

	log.Info("running server", zap.String("address", config.ServerAddress))
	var err error

	if config.EnableHTTPS {
		const (
			certFile = "servercert.crt"
			keyfile  = "servercert.key"
		)

		err = server.ListenAndServeTLS(certFile, keyfile)
	} else {
		err = server.ListenAndServe()
	}

	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err.Error(), zap.String(eventKey, "listen and serve"))
	}

	<-doneCh

	log.Info("server shutdown gracefully")
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
