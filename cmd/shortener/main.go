package main

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/pkg/errors"

	"github.com/nestjam/yap-shortener/internal/cert"
	conf "github.com/nestjam/yap-shortener/internal/config"
	env "github.com/nestjam/yap-shortener/internal/config/environment"
	"github.com/nestjam/yap-shortener/internal/domain"
	factory "github.com/nestjam/yap-shortener/internal/factory"
	httpserver "github.com/nestjam/yap-shortener/internal/server"
	grpcserver "github.com/nestjam/yap-shortener/internal/server/grpc"
	pb "github.com/nestjam/yap-shortener/proto"
)

const (
	certFile            = "servercert.crt"
	keyfile             = "servercert.key"
	shortenURLsMaxCount = 1000
	address             = "address"
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

	urlRemover := httpserver.NewURLRemover(ctx, doneCh, store, logger)

	if config.EnableHTTPS && (!exists(certFile) || !exists(keyfile)) {
		if err := generateAndSave(certFile, keyfile); err != nil {
			logger.Fatal(err.Error())
		}
	}

	httpServer := newHTTPServer(config, store, urlRemover, logger)

	grpcServer := newGRPCServer(config, store, urlRemover)
	tcpListener, err := net.Listen("tcp", ":3200")
	if err != nil {
		logger.Fatal(err.Error())
	}

	go func() {
		logger.Info("running http server", zap.String(address, config.ServerAddress))

		var err error
		if config.EnableHTTPS {
			err = httpServer.ListenAndServeTLS(certFile, keyfile)
		} else {
			err = httpServer.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			logger.Error(err.Error())
			cancel()
		}
	}()

	go func() {
		logger.Info("running grpc server", zap.String(address, tcpListener.Addr().String()))

		if err := grpcServer.Serve(tcpListener); err != nil {
			logger.Error(err.Error())
			cancel()
		}
	}()

	go func() {
		<-ctx.Done()

		if err := httpServer.Shutdown(context.Background()); err != nil {
			logger.Sugar().Infof("http server shut down: %v", err)
		}

		close(doneCh)
	}()

	<-doneCh

	logger.Info("servers shutdown gracefully")
}

func newGRPCServer(c conf.Config, store domain.URLStore, remover *httpserver.URLRemover) *grpc.Server {
	s := grpc.NewServer()

	pb.RegisterShortenerServer(s, grpcserver.New(store, c.BaseURL,
		grpcserver.WithShortenURLsMaxCount(shortenURLsMaxCount),
		grpcserver.WithURLsRemover(remover)))

	return s
}

func newHTTPServer(c conf.Config, store domain.URLStore, remover *httpserver.URLRemover, log *zap.Logger) *http.Server {
	handler := httpserver.New(store, c.BaseURL,
		httpserver.WithLogger(log),
		httpserver.WithShortenURLsMaxCount(shortenURLsMaxCount),
		httpserver.WithURLsRemover(remover),
		httpserver.WithTrustedSubnet(c.TrustedSubnet))

	return &http.Server{
		Addr:    c.ServerAddress,
		Handler: handler,
	}
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

func generateAndSave(certFile, keyfile string) error {
	const op = "generate and save"
	cert, key, err := cert.Generate()
	if err != nil {
		return errors.Wrap(err, op)
	}

	if err = writeFile(certFile, cert); err != nil {
		return errors.Wrap(err, op)
	}

	if err = writeFile(keyfile, key); err != nil {
		return errors.Wrap(err, op)
	}

	return nil
}

func writeFile(name string, data bytes.Buffer) error {
	const perm os.FileMode = 0600
	err := os.WriteFile(name, data.Bytes(), perm)
	if err != nil {
		return errors.Wrap(err, "write file")
	}

	return nil
}

func exists(file string) bool {
	_, err := os.Stat(file)
	return err == nil
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
