package kernel

import (
	"context"
	"sync"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-core/pkg/log"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"google.golang.org/grpc"
)

var chainClient iotexapi.APIServiceClient
var chainClientOnce sync.Once

func GetChainClient() iotexapi.APIServiceClient {
	chainClientOnce.Do(func() {
		conn1, err := grpc.DialContext(context.Background(), config.Default.Iotex.ChainEndPoint, grpc.WithBlock(), grpc.WithInsecure())
		if err != nil {
			log.L().Error("Failed to connect to chain's API server.")
		}
		chainClient = iotexapi.NewAPIServiceClient(conn1)
	})
	return chainClient
}
