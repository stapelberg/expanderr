package main

import (
	"lib"
	"log"
)

func logic() error {
	i := lib.Logic()
}

func main() {
	if err := logic(); err != nil {
		log.Fatal(err)
	}
}
