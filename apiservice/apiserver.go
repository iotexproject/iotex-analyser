package apiservice

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/iotexproject/iotex-analyser/api"
	"github.com/iotexproject/iotex-analyser/config"
	"github.com/soheilhy/cmux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func RegisterAPIService(grpcServer *grpc.Server) {
	api.RegisterAccountServiceServer(grpcServer, &AccountService{})
}

func StartGRPCServiceWithProxy() error {

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", config.Default.Server.GrpcPort))
	if err != nil {
		return err
	}
	m := cmux.New(l)
	mux := http.NewServeMux()
	gwmux := runtime.NewServeMux()
	mux.Handle("/", gwmux)
	grpcServer := grpc.NewServer()
	ctx := context.Background()
	//api.RegisterAccountServiceServer(grpcServer, &AccountService{})
	api.RegisterAccountServiceHandlerServer(ctx, gwmux, &AccountService{})
	//RegisterAPIService(grpcServer)
	reflection.Register(grpcServer)
	grpcL := m.Match(cmux.HTTP2HeaderField("content-type", "application/grpc"))
	httpL := m.Match(cmux.HTTP1Fast())
	go func() {
		if err := grpcServer.Serve(grpcL); err != nil {
			log.Panic(err)
		}
	}()

	httpS := &http.Server{
		Handler: mux,
	}
	go func() {
		if err := httpS.Serve(httpL); err != nil {
			log.Panic(err)
		}
	}()

	return m.Serve()
}
