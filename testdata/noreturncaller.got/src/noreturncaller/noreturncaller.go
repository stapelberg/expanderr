package main

import (
	"log"
	"os"
)

func logic() {
	os.Remove("/tmp/foo")
}

func main() {
	log.Printf("ohai")
	logic()
}
