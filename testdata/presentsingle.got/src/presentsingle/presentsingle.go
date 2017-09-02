package main

import (
	"log"
	"os"
)

func logic() (int, error) {
	err := os.Remove("a")
	return 0, nil
}

func main() {
	if _, err := logic(); err != nil {
		log.Fatal(err)
	}
}
