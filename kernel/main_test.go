package kernel

import (
	"os"
	"testing"

	"github.com/iotexproject/iotex-analyser/config"
)

// TestMain initializes the package-level rolldpos protocol before any test
// runs.
//
// Without it, rolldposProtocol is nil and the first epoch helper called from a
// test (NumSubEpochs, GetEpochNum, ...) dereferences nil. That is not merely a
// failing test: the panic tears down the whole test binary, so every test
// declared after epoch_test.go in filename order -- reward_test.go and
// staking_event_test.go -- never runs and is silently reported as absent rather
// than failed. `go test ./kernel/` printed "0 passed, 3 failed" while four
// reward tests sat in the package untouched.
//
// config.Default.Genesis is genesis.Default, which is what the production
// Init() path also starts from, so the protocol built here matches what the
// binary uses.
func TestMain(m *testing.M) {
	Init(&config.Default)
	os.Exit(m.Run())
}
