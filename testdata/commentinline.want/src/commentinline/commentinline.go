package main

import (
	"log"
	"os"
)

func logic() (int, error) {
	if err := os.Remove("/tmp/foo" /*path*/); err != nil {
		return 0, err
	} // delete
	return 0, nil
}

func main() {
	if _, err := logic(); err != nil {
		log.Fatal(err)
	}
}
