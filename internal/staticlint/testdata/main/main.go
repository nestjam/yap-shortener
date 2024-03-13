package main

import (
	"os"
	"syscall"
)

func main() {
	bar()
	Exit()
	syscall.Exit(1)

	os.Exit(1) // want "using exit in main"
}

// Exit - тестовый метод.
func Exit() {
}
