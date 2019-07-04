package main

import (
	"fmt"
	"io/ioutil"
)

func logic() (int, string) {
	b := ioutil.ReadAll(nil)
	return len(b), "hoi"
}

func main() {
	myInt, myStr := logic()
	fmt.Printf("%s, my number is %d", myStr, myInt)
}
