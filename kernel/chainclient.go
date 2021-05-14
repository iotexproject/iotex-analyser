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
)

var chainClient iotexapi.APIServiceClient
var chainClientOnce sync.Once

func ChainClient() iotexapi.APIServiceClient {
	chainClientOnce.Do(func() {
		var opt grpc.DialOption
		if !config.Default.Iotex.ChainInsecure {
			opt = grpc.WithInsecure()
		} else {
			opt = grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{}))
		}
		conn, err := grpc.Dial(config.Default.Iotex.ChainEndPoint, opt)
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

var DefaultHTTPClient = &http.Client{
	Transport: &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
	},
	Timeout: time.Second * 10,
}
