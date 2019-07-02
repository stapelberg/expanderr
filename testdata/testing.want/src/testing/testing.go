package main

import (
	"log"
	"os"
)

func logic() {
	if err := os.Remove("hello"); err != nil {
		panic(err)
	}
}

func logicTwo() int {
	if err := os.Remove("hello"); err != nil {
		panic(err)
	}
	return 0
}

func logicThree() {
	if _, err := os.Create("hello"); err != nil {
		panic(err)
	}
}

func logicFour() int {
	if _, err := os.Create("hello"); err != nil {
		panic(err)
	}
	return 0
}

func main() {
	log.Printf("ohai")
	logic()
}
