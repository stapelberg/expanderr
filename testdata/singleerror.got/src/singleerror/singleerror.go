package main

import (
	"log"
	"os"
)

func logic() (int, error) {
	os.Remove("/tmp/foo")
	return 0, nil
}

func main() {
	log.Printf("ohai")
	if _, err := logic(); err != nil {
		log.Fatal(err)
	}
}
