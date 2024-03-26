package main

import (
	. "os"
)

func main() {
	Exit(1) // want "using exit in main"
}
