package server

import (
	"context"
	"net"
	"net/rpc"
	"os"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/plugins"
)

type Server struct {
}

func New() *Server {
	return nil
}

// Start start the server
func (srv *Server) Start() error {
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
	rpc.Register(&RPC{})
	pluginService := plugins.NewService()
	for _, pluginFile := range config.Default.Server.Plugins {
		pluginArgs := &plugins.Args{Path: pluginFile}
		pluginReply := &plugins.Reply{}
		if err := pluginService.Load(pluginArgs, pluginReply); err != nil {
			return err
		}
	}
	rpc.Register(pluginService)
	rpc.Accept(listener)
	return nil
}

func (srv *Server) Stop(ctx context.Context) error {

	return nil
}
