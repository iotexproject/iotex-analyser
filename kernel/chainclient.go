package kernel

import (
	"crypto/tls"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-core/pkg/log"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var chainClient iotexapi.APIServiceClient
var chainClientOnce sync.Once

func ChainClient() iotexapi.APIServiceClient {
	chainClientOnce.Do(func() {
		var opt grpc.DialOption
		if !config.Default.Iotex.ChainInsecure {
			opt = grpc.WithTransportCredentials(insecure.NewCredentials())
		} else {
			opt = grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{}))
		}
		conn, err := grpc.Dial(config.Default.Iotex.ChainEndPoint, opt, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(32*10e6)))
		if err != nil {
			log.L().Error("failed to connect to chain endpoint.",
				zap.Error(err),
				zap.String("endpoint", config.Default.Iotex.ChainEndPoint),
			)
		}
		chainClient = iotexapi.NewAPIServiceClient(conn)
	})
	return chainClient
}

func ChainClientWithEndPoint(endpoint string, withInsecure bool) (iotexapi.APIServiceClient, error) {
	var opt grpc.DialOption
	if withInsecure {
		opt = grpc.WithTransportCredentials(insecure.NewCredentials())
	} else {
		opt = grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{}))
	}
	conn, err := grpc.Dial(endpoint, opt)
	if err != nil {
		return nil, err
	}
	return iotexapi.NewAPIServiceClient(conn), nil
}

var DefaultHTTPClient = &http.Client{
	Transport: &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
	},
	Timeout: time.Second * 10,
}
