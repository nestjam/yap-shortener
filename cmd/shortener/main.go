package main

import (
	"log"
	"net/http"
	"os"

	conf "github.com/nestjam/yap-shortener/internal/config"
	"github.com/nestjam/yap-shortener/internal/server"
	"github.com/nestjam/yap-shortener/internal/store"
)

func main() {
	config := conf.Parse(os.Args)
	store := store.NewInMemory()
	server := server.New(store, config.BaseAddr)
	log.Fatal(http.ListenAndServe(config.RunAddr, server))
}
