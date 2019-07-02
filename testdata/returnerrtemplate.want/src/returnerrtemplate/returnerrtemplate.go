package main

import (
	"fmt"
	"io/ioutil"
)

func logic() (int, string) {
	b, err := ioutil.ReadAll(nil)
	if err != nil {
		mylog.Fatal(err.Error())
		return 0, ""
	}
	return len(b), "hoi"
}

func main() {
	myInt, myStr := logic()
	fmt.Printf("%s, my number is %d", myStr, myInt)
}
