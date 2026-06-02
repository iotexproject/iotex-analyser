package main

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/iotexproject/iotex-core/v2/action"
)

func TestGetDataFromAction_TruncatesToSelector(t *testing.T) {
	const contract = "io1qnpz47hx5q6r3w876f9j7v2y8gqsm66nfmkn8u"
	// 36-byte payload: an arbitrary 4-byte selector + a 32-byte word.
	// The exact selector bytes don't matter; we only care that the
	// 4-byte prefix is preserved and the rest is dropped.
	full := []byte{
		0xa9, 0x05, 0x9c, 0xbb,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1,
	}

	cases := []struct {
		name             string
		in               []byte
		want             []byte
		requireNonNilOut bool // NOT NULL column: returned slice must be non-nil
	}{
		{"longer than selector keeps only selector", full, full[:4], true},
		{"exactly selector length is unchanged", full[:4], full[:4], true},
		{"shorter than selector is unchanged", []byte{0x12, 0x34}, []byte{0x12, 0x34}, true},
		{"nil normalised to empty slice", nil, []byte{}, true},
		{"empty slice preserved", []byte{}, []byte{}, true},
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
			if tc.requireNonNilOut && gotData == nil {
				t.Error("returned data is nil; column is NOT NULL so we must always return a non-nil slice")
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

// Regression: the truncated slice must not pin the original (potentially
// multi-KB) calldata buffer. We assert independence by mutating the source
// after the call and checking the returned slice is unchanged, and that
// its capacity matches its length (i.e. it owns its own small buffer).
func TestGetDataFromAction_DoesNotPinSourceBuffer(t *testing.T) {
	src := make([]byte, 1024)
	src[0], src[1], src[2], src[3] = 0xde, 0xad, 0xbe, 0xef
	ex := action.NewExecution("io1qnpz47hx5q6r3w876f9j7v2y8gqsm66nfmkn8u", big.NewInt(0), src)

	_, got, err := getDataFromAction(ex)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cap(got) > methodSelectorLen {
		t.Errorf("returned slice cap=%d > methodSelectorLen=%d: original backing array is being retained",
			cap(got), methodSelectorLen)
	}

	for i := range src {
		src[i] = 0xff
	}
	want := []byte{0xde, 0xad, 0xbe, 0xef}
	if !bytes.Equal(got, want) {
		t.Errorf("returned slice changed after mutating source: got %x, want %x", got, want)
	}
}
