package main

import (
	"testing"

	"github.com/iotexproject/iotex-analyser/config"
)

func TestERC1155SBT(t *testing.T) {
	initAddress()
	config.Default.Iotex.ChainEndPoint = "api.iotex.one:443"
	config.Default.Iotex.EVMNetworkID = 4689
	config.Default.Iotex.ChainInsecure = true
	ok, err := isSBT("io14v3wnklmrd3k4ul82950gm6n82m3pdlp2gzwxu", erc721ABI)
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Fatal("should be sbt")
	}
}
