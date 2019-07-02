package main

import (
	"log"
	"os"
)

func logic() {
	os.Remove("hello")
}

func logicTwo() error {
	foo := os.Remove("hello")
	return 0
}

func logicThree() {
	os.Create("hello")
}

func logicFour() int {
	os.Create("hello")
	return 0
}

func main() {
	log.Printf("ohai")
	logic()
	logicTwo()
}
