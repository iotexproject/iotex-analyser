package main

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/iotexproject/iotex-core/v2/action"
)

func TestGetDataFromAction_TruncatesToSelector(t *testing.T) {
	const contract = "io1qnpz47hx5q6r3w876f9j7v2y8gqsm66nfmkn8u"
	// 36-byte calldata: 4-byte selector + 32-byte uint256 argument.
	full := []byte{
		0xa9, 0x05, 0x9c, 0xbb, // transfer(address,uint256) selector
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1,
	}

	cases := []struct {
		name string
		in   []byte
		want []byte
	}{
		{"longer than selector keeps only selector", full, full[:4]},
		{"exactly selector length is unchanged", full[:4], full[:4]},
		{"shorter than selector is unchanged", []byte{0x12, 0x34}, []byte{0x12, 0x34}},
		{"nil normalised to empty slice", nil, []byte("")},
		{"empty slice preserved", []byte{}, []byte{}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ex := action.NewExecution(contract, big.NewInt(0), tc.in)
			gotContract, gotData, err := getDataFromAction(ex)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotContract != contract {
				t.Errorf("contract: got %q, want %q", gotContract, contract)
			}
			if !bytes.Equal(gotData, tc.want) {
				t.Errorf("data: got %x, want %x", gotData, tc.want)
			}
			if len(gotData) > methodSelectorLen {
				t.Errorf("data length %d exceeds methodSelectorLen %d", len(gotData), methodSelectorLen)
			}
		})
	}
}

func TestGetDataFromAction_NonExecutionReturnsError(t *testing.T) {
	tsf := action.NewTransfer(big.NewInt(0), "io1qnpz47hx5q6r3w876f9j7v2y8gqsm66nfmkn8u", nil)
	if _, _, err := getDataFromAction(tsf); err == nil {
		t.Fatal("expected error for non-Execution action, got nil")
	}
}
