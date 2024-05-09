package main

import (
	"fmt"
	"testing"
)

func TestName(t *testing.T) {
	if h, err := get(); err != nil {
		fmt.Println(err.Error())
	} else if h == 1 {
		fmt.Println(h)
	}
}

func get() (int, error) {
	return 1, nil
}
