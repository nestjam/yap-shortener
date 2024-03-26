package main

import (
	"context"
	"fmt"
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

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	printBuildInfo()

	config := getConfig()

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
