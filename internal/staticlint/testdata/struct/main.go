package main

import "os"

func main() {
	os.Exit(1) // want "using exit in main"

	os := foo{}
	os.Exit(1)
}

type foo struct {
}

// Exit - тестовый метод.
func (f foo) Exit(code int) {
}
