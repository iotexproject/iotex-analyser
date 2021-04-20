package apiservice

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/iotexproject/iotex-analyser/api"
	"github.com/iotexproject/iotex-analyser/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func registerAPIService(grpcServer *grpc.Server) {
	api.RegisterAccountServiceServer(grpcServer, &AccountService{})
}

func registerProxyAPIService(ctx context.Context, mux *runtime.ServeMux) error {
	if err := api.RegisterAccountServiceHandlerServer(ctx, mux, &AccountService{}); err != nil {
		return err
	}
	return nil
}

func StartGRPCService() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Default.Server.GrpcPort))
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer()
	registerAPIService(grpcServer)
	reflection.Register(grpcServer)
	return grpcServer.Serve(lis)
}

func StartGRPCProxyService() error {
	gwmux := runtime.NewServeMux()
	ctx := context.Background()
	if err := registerProxyAPIService(ctx, gwmux); err != nil {
		return err
	}
	port := fmt.Sprintf(":%d", config.Default.Server.GrpcProxyPort)
	gwServer := &http.Server{
		Addr:    port,
		Handler: gwmux,
	}
	return gwServer.ListenAndServe()
}
