package main

import (
	"log"
	"os"
)

func logic() (int, error) {
	f, ferr := os.Create("/tmp/a")
	if ferr != nil {
		return 0, ferr
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
