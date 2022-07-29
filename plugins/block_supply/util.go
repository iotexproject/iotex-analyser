package main

import (
	"database/sql"
	"errors"
	"math/big"

	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
)

const (
	lockAddresses = "io1uqhmnttmv0pg8prugxxn7d8ex9angrvfjfthxa" // Separate multiple addresses with ","
	totalBalance  = "12700000000000000000000000000"             // 10B + 2.7B (due to Postmortem 1)
	nsv1Balance   = "262281303940000000000000000"
	bnfxBalance   = "3414253030000000000000000"
)

func accountBalanceByHeight(height uint64, addr string) (*big.Int, error) {
	db := db.DB()
	var amount sql.NullString
	query := "SELECT sum(in_flow)-sum(out_flow) from account_income WHERE block_height<=? and address=?"
	err := db.Raw(query, height, addr).Scan(&amount).Error
	if err != nil {
		return nil, err
	}
	balance, ok := big.NewInt(0).SetString(amount.String, 10)
	if !ok {
		balance = big.NewInt(0)
	}

	return balance, nil
}
func getTotalSupply(height uint64) (string, error) {
	// get zero address balance.
	zeroAddressBalance, err := accountBalanceByHeight(height, address.ZeroAddress)
	if err != nil {
		return "", err
	}

	// Convert string format to big.Int format
	totalBalanceInt, _ := new(big.Int).SetString(totalBalance, 10)
	nsv1BalanceInt, _ := new(big.Int).SetString(nsv1Balance, 10)
	bnfxBalanceInt, _ := new(big.Int).SetString(bnfxBalance, 10)

	// Compute 10B + 2.7B (due to Postmortem 1) - Balance(all zero address) - Balance(nsv1) - Balance(bnfx)
	return new(big.Int).Sub(new(big.Int).Sub(new(big.Int).Sub(totalBalanceInt, zeroAddressBalance), nsv1BalanceInt), bnfxBalanceInt).String(), nil
}

func getTotalCirculatingSupply(height uint64, totalSupply string) (string, error) {
	locked, err := accountBalanceByHeight(height, lockAddresses)
	if err != nil {
		return "", err
	}

	totalSupplyBig, ok := new(big.Int).SetString(totalSupply, 10)
	if !ok {
		return "", errors.New("failed to format to big int:" + totalSupply)
	}

	return new(big.Int).Sub(totalSupplyBig, locked).String(), nil
}
