package kernel

import (
	"sync"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var (
	ethArchiveClient     *ethclient.Client
	ethArchiveClientErr  error
	ethArchiveClientOnce sync.Once
)

// EthArchiveClient returns a singleton eth-json-rpc client backed by the
// EthArchiveEndPoint config. The endpoint must be an archive node so that
// state-at-height queries (eth_getCode, eth_getTransactionCount) work for
// historical blocks.
func EthArchiveClient() (*ethclient.Client, error) {
	ethArchiveClientOnce.Do(func() {
		endpoint := config.Default.Iotex.EthArchiveEndPoint
		if endpoint == "" {
			ethArchiveClientErr = errors.New("iotex.ethArchiveEndPoint is not configured")
			return
		}
		c, err := ethclient.Dial(endpoint)
		if err != nil {
			log.L().Error("failed to dial eth archive endpoint",
				zap.Error(err),
				zap.String("endpoint", endpoint),
			)
			ethArchiveClientErr = errors.Wrapf(err, "dial %s", endpoint)
			return
		}
		ethArchiveClient = c
	})
	return ethArchiveClient, ethArchiveClientErr
}
