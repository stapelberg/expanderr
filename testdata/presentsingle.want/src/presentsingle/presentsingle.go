package main

import (
	"log"
	"os"
)

func logic() (int, error) {
	if err := os.Remove("a"); err != nil {
		return 0, err
	}
	return 0, nil
}

func main() {
	if _, err := logic(); err != nil {
		log.Fatal(err)
	}
}
