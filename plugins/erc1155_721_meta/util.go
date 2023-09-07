package main

import (
	"encoding/hex"
	"strings"

	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/pkg/errors"
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

func isErc1155(addr string) (bool, error) {
	if _, ok := nonErc1155Contract[addr]; ok {
		return false, nil
	}
	if _, ok := erc1155Contract[addr]; !ok {
		ok, err := kernel.IsErc1155(addr)
		if err != nil {
			return false, errors.Wrap(err, "failed to check erc1155")
		}
		if !ok {
			nonErc1155Contract[addr] = struct{}{}
			return false, nil
		}
		erc1155Contract[addr] = struct{}{}
	}
	return true, nil
}
