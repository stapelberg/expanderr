package main

import (
	"log"
	"os"
)

func logic() (int, error) {
	f, err := os.Create("/tmp/a")
	if err != nil {
		return 0, err
	}
	if _, err := f.Write([]byte("foo")); err != nil {
		return 0, err
	}
	return 0, nil
}

func main() {
	if _, err := logic(); err != nil {
		log.Fatal(err)
	}
}
