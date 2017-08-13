package main

import (
	"lib"
	"log"
)

func logic() error {
	i, err := lib.Logic()
	if err != nil {
		return err
	}
}

func main() {
	if err := logic(); err != nil {
		log.Fatal(err)
	}
}
