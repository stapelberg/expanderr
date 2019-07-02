package main

import (
	"log"
	"os"
)

func logic() {
	if err := os.Remove("/tmp/foo"); err != nil {
		panic(err)
	}
}

func main() {
	log.Printf("ohai")
	logic()
}
