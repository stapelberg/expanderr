package main

import (
	"log"
)

var boom func() error

func testing() error {
	boom()
	return nil
}

func main() {
	if err := testing(); err != nil {
		log.Fatal(err.Error())
	}
}
