package main

import (
	"log"
	"os"
)

func logic() int {
	os.Remove("/tmp/foo")
	return 42
}

func main() {
	myInt := logic()
	log.Printf("ohai %d", myInt)
}
