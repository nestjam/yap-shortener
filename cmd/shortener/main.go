package main

import (
	"log"
	"net/http"
	"os"

	conf "github.com/nestjam/yap-shortener/internal/config"
	env "github.com/nestjam/yap-shortener/internal/config/environment"
	"github.com/nestjam/yap-shortener/internal/server"
	"github.com/nestjam/yap-shortener/internal/store"
)

func main() {
	config := conf.New().
		FromArgs(os.Args).
		FromEnv(env.New())
	store := store.NewInMemory()
	server := server.New(store, config.BaseURL)
	log.Fatal(http.ListenAndServe(config.ServerAddress, server))
}
