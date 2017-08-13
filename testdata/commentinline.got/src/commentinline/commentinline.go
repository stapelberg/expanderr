package main

import (
	"log"
	"os"
)

func logic() (int, error) {
	os.Remove("/tmp/foo" /*path*/) // delete
	return 0, nil
}

func main() {
	if _, err := logic(); err != nil {
		log.Fatal(err)
	}
}
