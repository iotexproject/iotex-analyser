package server

import (
	"context"
	"net"
	"net/rpc"
	"os"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/plugins"
	"github.com/iotexproject/iotex-core/pkg/log"
	"github.com/pkg/errors"
)

type Server struct {
}

func New() *Server {
	s := &Server{}
	return s
}

// Start start the server
func (srv *Server) Start() error {
	if err := db.GetDB().Ping(); err != nil {
		return errors.Wrap(err, "failed to ping DB")
	}
	log.L().Info("start RPC service")
	if err := startRPCService(); err != nil {
		return errors.Wrap(err, "failed to start RPC service")
	}
	log.L().Info("started RPC service")
	return nil
}

func (srv *Server) Stop(ctx context.Context) error {

	return nil
}

func startRPCService() error {
	sockAddr := config.Default.Server.Addr
	os.Remove(sockAddr)
	unixAddr, err := net.ResolveUnixAddr("unix", sockAddr)
	if err != nil {
		return err
	}

	listener, err := net.ListenUnix("unix", unixAddr)
	if err != nil {
		return err
	}

	pluginService := plugins.NewService()
	for _, pluginFile := range config.Default.Server.Plugins {
		pluginArgs := &plugins.Args{Path: pluginFile}
		pluginReply := &plugins.Reply{}
		if err := pluginService.Load(pluginArgs, pluginReply); err != nil {
			return err
		}
	}
	if err := rpc.Register(pluginService); err != nil {
		return err
	}
	rpc.Accept(listener)
	return nil
}
