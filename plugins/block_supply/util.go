package main

import (
	"database/sql"
	"errors"
	"math/big"

	"github.com/iotexproject/iotex-analyser/db"
)

const (
	lockAddresses = "io1uqhmnttmv0pg8prugxxn7d8ex9angrvfjfthxa" // Separate multiple addresses with ","
	totalBalance  = "12700000000000000000000000000"             // 10B + 2.7B (due to Postmortem 1)
	nsv1Balance   = "262281303940000000000000000"
	bnfxBalance   = "3414253030000000000000000"
)

// accountBalanceByHeight returns the cumulative net balance (sum(in_flow)-sum(out_flow))
// for addr up to and including height.  Called once during Start() to seed runningBalance.
func accountBalanceByHeight(height uint64, addr string) (*big.Int, error) {
	gormDB := db.DB()
	var amount sql.NullString
	err := gormDB.Raw(
		"SELECT sum(in_flow)-sum(out_flow) FROM account_income WHERE block_height<=? AND address=?",
		height, addr,
	).Scan(&amount).Error
	if err != nil {
		return nil, err
	}
	balance, ok := big.NewInt(0).SetString(amount.String, 10)
	if !ok {
		balance = big.NewInt(0)
	}
	return balance, nil
}

// accountBalanceDeltaAtHeight returns the net flow (in_flow - out_flow) for addr
// at exactly height — a single-block point query used in putBlock.
func accountBalanceDeltaAtHeight(height uint64, addr string) (*big.Int, error) {
	gormDB := db.DB()
	var amount sql.NullString
	err := gormDB.Raw(
		"SELECT sum(in_flow)-sum(out_flow) FROM account_income WHERE block_height=? AND address=?",
		height, addr,
	).Scan(&amount).Error
	if err != nil {
		return nil, err
	}
	delta, ok := big.NewInt(0).SetString(amount.String, 10)
	if !ok {
		delta = big.NewInt(0)
	}
	return delta, nil
}

// computeTotalSupply derives total supply from the in-memory zero-address balance.
// No DB access needed.
func computeTotalSupply(zeroAddrBal *big.Int) string {
	totalBalanceInt, _ := new(big.Int).SetString(totalBalance, 10)
	nsv1BalanceInt, _ := new(big.Int).SetString(nsv1Balance, 10)
	bnfxBalanceInt, _ := new(big.Int).SetString(bnfxBalance, 10)
	// 10B + 2.7B (Postmortem 1) − zeroAddr − nsv1 − bnfx
	return new(big.Int).Sub(
		new(big.Int).Sub(
			new(big.Int).Sub(totalBalanceInt, zeroAddrBal),
			nsv1BalanceInt,
		),
		bnfxBalanceInt,
	).String()
}

// computeTotalCirculatingSupply derives circulating supply from the in-memory locked balance.
// No DB access needed.
func computeTotalCirculatingSupply(totalSupply string, lockedBal *big.Int) (string, error) {
	totalSupplyBig, ok := new(big.Int).SetString(totalSupply, 10)
	if !ok {
		return "", errors.New("failed to format to big int:" + totalSupply)
	}
	return new(big.Int).Sub(totalSupplyBig, lockedBal).String(), nil
}
