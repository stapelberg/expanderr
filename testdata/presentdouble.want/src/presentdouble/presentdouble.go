package main

import (
	"io/ioutil"
	"log"
)

func logic() (int, error) {
	n, err := ioutil.ReadAll(nil)
	if err != nil {
		return 0, err
	}
	return 0, nil
}

func main() {
	if _, err := logic(); err != nil {
		log.Fatal(err)
	}
}
