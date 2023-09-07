package main

import (
	"encoding/hex"
	"strings"
)

func safeHexDecode(data string) []byte {
	if data == "" {
		panic("data is empty")
	}
	data = strings.TrimPrefix(data, "0x")
	decoded, err := hex.DecodeString(data)
	if err != nil {
		panic(err)
	}
	return decoded
}
