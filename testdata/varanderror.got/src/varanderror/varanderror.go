package main

import (
	"io"
	"io/ioutil"
	"log"
	"strings"
)

func logic() (int, error) {
	n := io.Copy(ioutil.Discard, strings.NewReader("test"))
	return 0, nil
}

func main() {
	log.Printf("ohai")
	if _, err := logic(); err != nil {
		log.Fatal(err)
	}
}
