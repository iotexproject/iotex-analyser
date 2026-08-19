package voter_reward

import (
	"encoding/hex"
	"math/big"

	"github.com/iotexproject/iotex-address/address"
	"github.com/shopspring/decimal"
)

// addressString renders raw address bytes as bech32, tolerating the empty
// slice the protocol writes for an unset address.
func addressString(b []byte) (string, error) {
	if len(b) == 0 {
		return "", nil
	}
	addr, err := address.FromBytes(b)
	if err != nil {
		return "", err
	}
	return addr.String(), nil
}

// dec converts a possibly-nil big.Int to the decimal type the models use.
func dec(v *big.Int) decimal.Decimal {
	if v == nil {
		return decimal.Zero
	}
	return decimal.NewFromBigInt(v, 0)
}

// bytesToDec reads a big-endian unsigned amount as stored in the protobuf
// allocation records.
func bytesToDec(b []byte) decimal.Decimal {
	if len(b) == 0 {
		return decimal.Zero
	}
	return decimal.NewFromBigInt(new(big.Int).SetBytes(b), 0)
}

// hexOrEmpty renders raw cursor bytes the same way the protocol's own
// CURSOR_PROGRESS log does (%x), so a value read from state and a value parsed
// from a log are directly comparable.
func hexOrEmpty(b []byte) (string, error) {
	if len(b) == 0 {
		return "", nil
	}
	return hex.EncodeToString(b), nil
}
