package main

import (
	"log"
	"net/http"

	shortener "github.com/nestjam/yap-shortener/internal/server"
	"github.com/nestjam/yap-shortener/internal/store"
)

func main() {
	store := store.NewInMemory()
	log.Fatal(http.ListenAndServe(":8080", shortener.New(store)))
}
