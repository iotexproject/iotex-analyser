package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"
)

func TestERC1155URI(t *testing.T) {
	initAddress()
	a, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000001868747470733a2f2f6170692e736f746164782e636f6d2f380000000000000000")
	if err != nil {
		t.Fatal(err)
	}
	aa, err := erc1155ABI.Unpack("uri", a)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%v", aa)
	fmt.Printf("\naaaa %x aaa\n", bytes.TrimSpace(a))
}
