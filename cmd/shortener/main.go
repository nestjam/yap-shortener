package main

import (
	"log"
	"net/http"

	"github.com/nestjam/yap-shortener/internal/app"
)

func main() {
	s := app.NewShortenerServer()
	handler := http.HandlerFunc(s.ServeHTTP)
	log.Fatal(http.ListenAndServe(":8080", handler))
}
