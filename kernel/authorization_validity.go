package kernel

import (
	"bytes"
	"context"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/iotexproject/iotex-analyser/config"
	slog "github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	validityMaxRetries = 3
	validityRetryDelay = 200 * time.Millisecond
)

// delegationDesignatorPrefix is the EIP-7702 magic prefix that an authority's
// code starts with after a delegation has been installed.
var delegationDesignatorPrefix = []byte{0xef, 0x01, 0x00}

// ComputeAuthorizationValidity evaluates whether an EIP-7702 authorization
// would have been accepted by the chain at the time it was included.
//
// Validity rules (mirroring iotex-core's validateAuthorization):
//  1. chain_id is 0 or matches the network chain id
//  2. authority's code at blockHeight-1 is empty OR a delegation designator
//  3. authority's nonce at blockHeight-1 equals authNonce
//
// The blacklist rule is intentionally skipped. Signature recovery is
// the caller's responsibility (the recovered authority is an input).
//
// Returns:
//   - (true, nil) if all rules pass
//   - (false, nil) if any rule failed for a deterministic reason
//   - (false, err) only when RPC could not reach a decision after retries
func ComputeAuthorizationValidity(
	ctx context.Context,
	authority common.Address,
	blockHeight uint64,
	chainID *uint256.Int,
	authNonce uint64,
) (bool, error) {
	// Rule 1: chain_id must be 0 or match network chain id.
	networkChainID := uint64(config.Default.Iotex.EVMNetworkID)
	if chainID != nil && !chainID.IsZero() && chainID.Uint64() != networkChainID {
		return false, nil
	}

	// State-at-height: state at the END of (blockHeight-1) is the state
	// at the START of the current block. For block 0 (genesis) fall back to nil
	// which the RPC treats as latest — but genesis contains no txs anyway.
	var preBlockNum *big.Int
	if blockHeight > 0 {
		preBlockNum = new(big.Int).SetUint64(blockHeight - 1)
	}

	// Rule 2: authority's code must be empty or a delegation designator.
	code, err := getCodeAtWithRetry(ctx, authority, preBlockNum)
	if err != nil {
		return false, err
	}
	if len(code) != 0 && !bytes.HasPrefix(code, delegationDesignatorPrefix) {
		return false, nil
	}

	// Rule 3: authority's nonce must equal authNonce.
	nonce, err := getNonceAtWithRetry(ctx, authority, preBlockNum)
	if err != nil {
		return false, err
	}
	if nonce != authNonce {
		return false, nil
	}

	return true, nil
}

func getCodeAtWithRetry(ctx context.Context, addr common.Address, blockNum *big.Int) ([]byte, error) {
	client, err := EthArchiveClient()
	if err != nil {
		return nil, err
	}
	var lastErr error
	for attempt := 0; attempt < validityMaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(validityRetryDelay):
			}
		}
		code, err := client.CodeAt(ctx, addr, blockNum)
		if err == nil {
			return code, nil
		}
		lastErr = err
		slog.L().Warn("eth_getCode failed, will retry",
			zap.String("address", addr.Hex()),
			zap.Any("block", blockNum),
			zap.Int("attempt", attempt+1),
			zap.Error(err),
		)
	}
	return nil, errors.Wrapf(lastErr, "eth_getCode(%s, %v) after %d retries", addr.Hex(), blockNum, validityMaxRetries)
}

func getNonceAtWithRetry(ctx context.Context, addr common.Address, blockNum *big.Int) (uint64, error) {
	client, err := EthArchiveClient()
	if err != nil {
		return 0, err
	}
	var lastErr error
	for attempt := 0; attempt < validityMaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-time.After(validityRetryDelay):
			}
		}
		nonce, err := client.NonceAt(ctx, addr, blockNum)
		if err == nil {
			return nonce, nil
		}
		lastErr = err
		slog.L().Warn("eth_getTransactionCount failed, will retry",
			zap.String("address", addr.Hex()),
			zap.Any("block", blockNum),
			zap.Int("attempt", attempt+1),
			zap.Error(err),
		)
	}
	return 0, errors.Wrapf(lastErr, "eth_getTransactionCount(%s, %v) after %d retries", addr.Hex(), blockNum, validityMaxRetries)
}
