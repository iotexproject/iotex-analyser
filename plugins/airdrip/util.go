package main

import (
	"encoding/hex"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
)

func getAddresFromHash256(h hash.Hash256) (address.Address, error) {
	hexStr := hex.EncodeToString(h[:])
	ethAddr := hexStr[24:]
	ethAddress := common.HexToAddress(ethAddr)
	return address.FromBytes(ethAddress.Bytes())
}

func getAddressFromHash256ByIndex(h []hash.Hash256, i int) (address.Address, error) {
	if i >= len(h) {
		return nil, errors.New("invalid index")
	}
	return getAddresFromHash256(h[i])
}

func getbigIntFromHash256ByIndex(h []hash.Hash256, i int) (*big.Int, error) {
	if i >= len(h) {
		return nil, errors.New("invalid index")
	}
	return new(big.Int).SetBytes(h[i][:]), nil
}
