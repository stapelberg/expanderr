package main

import (
	"log"
	"os"
)

func logic() error {
	os.Clearenv()
}

func main() {
	if err := logic(); err != nil {
		log.Fatal(err)
	}
}
