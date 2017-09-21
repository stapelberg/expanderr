package err

import "os"

type customstr string

type customstruct struct{}

type custominterface interface{}

func err() (customstr, customstruct, custominterface, error) {
	a := os.Getwd()
	return nil
}
