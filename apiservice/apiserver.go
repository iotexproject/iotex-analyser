package apiservice

import (
	"github.com/iotexproject/iotex-analyser/api"
	"google.golang.org/grpc"
)

func RegisterAPIService(grpcServer *grpc.Server) {
	api.RegisterAccountServiceServer(grpcServer, &AccountService{})
}
