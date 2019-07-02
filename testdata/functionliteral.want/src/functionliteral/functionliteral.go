package main

import (
	"log"
)

var boom func() error

func testing() error {
	if err := boom(); err != nil {
		return err
	}
	return nil
}

func main() {
	if err := testing(); err != nil {
		log.Fatal(err.Error())
	}
}
