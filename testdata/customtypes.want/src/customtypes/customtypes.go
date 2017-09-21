package err

import "os"

type customstr string

type customstruct struct{}

type custominterface interface{}

func err() (customstr, customstruct, custominterface, error) {
	a, err := os.Getwd()
	if err != nil {
		return "", customstruct{}, nil, err
	}
	return nil
}
