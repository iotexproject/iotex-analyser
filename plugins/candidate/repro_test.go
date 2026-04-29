// Reproduce the production candidate plugin failure observed on testnet at
// block 43063855 by running handleBlock against the live testnet PG and
// live testnet RPC. Without the operator fallback this test fails at the
// CandidateUpdate signed by the (newly-set) operator with
// `record not found`; with the fallback it processes the whole batch.
//
// Skipped unless LIVE_TESTNET_DSN env var is set; the inner work is wrapped in
// a transaction that always rolls back, so it does not mutate live data.
//
// Usage:
//
//	LIVE_TESTNET_DSN='host=… port=… user=… password=… dbname=testnet sslmode=disable' \
//	    go test ./plugins/candidate/ -run TestReproduce -count=1 -v
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/iotexproject/iotex-analyser/config"
	dbpkg "github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func fmtParseUint32(s string, dst *uint32) (int, error) {
	return fmt.Sscanf(s, "%d", dst)
}

const (
	// 43063855 is the actual failing block: a CandidateUpdate signed by
	// io17qs77wkufe66… which the chain accepted because that address had
	// just been set as the candidate's *operator* at block 43063850. The
	// pre-fix plugin fetched only by `owner_address = sender` and missed it.
	failingHeight = 43063855
	failingSender = "io17qs77wkufe66gv7r2m9gq3naxr64zxat0yjvzp"
)

// errRollback is returned from inside the gorm Transaction closure to force a
// rollback after we've observed the handleBlock outcome. handleBlock's own
// error (if any) is captured separately so we report the real failure mode.
var errRollback = errors.New("test rollback (intentional)")

func TestReproduceBlock43063796(t *testing.T) {
	dsn := os.Getenv("LIVE_TESTNET_DSN")
	if dsn == "" {
		t.Skip("LIVE_TESTNET_DSN env not set; this test only runs when explicitly enabled")
	}
	endpoint := os.Getenv("LIVE_CHAIN_ENDPOINT")
	if endpoint == "" {
		endpoint = "api.testnet.iotex.one:443"
	}
	insecureStr := os.Getenv("LIVE_CHAIN_INSECURE")
	chainInsecure := insecureStr == "" || insecureStr == "1" || insecureStr == "true"

	// 1. config — kernel.ChainClient() reads from config.Default.
	config.Default.Iotex.ChainEndPoint = endpoint
	config.Default.Iotex.ChainInsecure = chainInsecure
	// EVMNetworkID is required by block.NewDeserializer (called from
	// GetBlocksFromChain). 4690 = IoTeX testnet chain ID.
	if id := os.Getenv("LIVE_EVM_NETWORK_ID"); id != "" {
		var n uint32
		_, _ = fmtParseUint32(id, &n)
		config.SetEVMNetworkID(n)
	} else {
		config.SetEVMNetworkID(4690)
	}

	// 2. PG.
	gormLogger := logger.New(
		log.New(os.Stdout, "[gorm] ", log.LstdFlags),
		logger.Config{SlowThreshold: 200 * time.Millisecond, LogLevel: logger.Info, Colorful: false},
	)
	pg, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormLogger})
	require.NoError(t, err, "open testnet pg")
	dbpkg.SetDB(pg)

	// 3. Sanity: confirm we can reach the candidate table on the live
	//    connection at all. We deliberately avoid asserting the specific
	//    state here, because the test runs handleBlock against blocks that
	//    the production analyser has not yet processed (it's stuck), so
	//    the live state will be missing the operator switch that occurs
	//    inside the test's own (rolled-back) tx at block 43063850.
	var rowCount int64
	require.NoError(t, pg.Raw("SELECT count(*) FROM candidate").Scan(&rowCount).Error)
	require.Greater(t, rowCount, int64(0), "candidate table should be non-empty on testnet")

	// 4. Fetch the batch from chain RPC. Production failed at
	//    height=43063723, batchSize=512. We use a smaller batch (start to
	//    just past the failing height) to keep the test cheap while still
	//    matching the multi-block tx shape.
	const batchStart = 43063723
	const batchCount = 150 // covers up through 43063872 — past the failing 43063855
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	blks, err := kernel.GetBlocksFromChain(ctx, batchStart, batchCount, kernel.ChainClient())
	require.NoError(t, err, "fetch batch [%d, +%d) via grpc", batchStart, batchCount)
	require.Len(t, blks, batchCount)
	t.Logf("fetched %d blocks: %d..%d", len(blks), blks[0].Height(), blks[len(blks)-1].Height())

	// 5. Run handleBlock for each block inside a single tx (mimicking
	//    PutBlocks behavior introduced in PR #135), then roll back.
	var handleErr error
	var failedHeight uint64
	txErr := pg.Transaction(func(tx *gorm.DB) error {
		for _, blk := range blks {
			if err := handleBlock(blk, tx); err != nil {
				handleErr = err
				failedHeight = blk.Height()
				break
			}
		}
		// Always roll back so this test never persists data.
		return errRollback
	})
	require.ErrorIs(t, txErr, errRollback, "tx must roll back, got %v", txErr)

	if handleErr == nil {
		t.Logf("all %d blocks (%d..%d) SUCCEEDED — fix is effective",
			len(blks), blks[0].Height(), blks[len(blks)-1].Height())
		return
	}
	if errors.Is(handleErr, gorm.ErrRecordNotFound) {
		t.Fatalf("REGRESSION: handleBlock returned ErrRecordNotFound at height %d — operator-fallback fix appears not to be applied", failedHeight)
	}
	t.Fatalf("handleBlock returned an unexpected error at height %d: %v", failedHeight, handleErr)
}
