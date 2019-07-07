package main

import (
	"log"
	"os"
)

func logic() int {
	if err := os.Remove("/tmp/foo"); err != nil {
		panic(err)
	}
	return 42
}

func main() {
	myInt := logic()
	log.Printf("ohai %d", myInt)
}
