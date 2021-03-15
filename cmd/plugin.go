package cmd

import (
	"fmt"
	"net/rpc"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/plugins"
	"github.com/iotexproject/iotex-analyser/server"
	"github.com/iotexproject/iotex-core/pkg/log"
	"github.com/urfave/cli/v2"
)

var Plugin = &cli.Command{
	Name:        "plugin",
	Usage:       "start plugin",
	Description: `start plugin`,
	Action:      runPlugin,
}

func runPlugin(c *cli.Context) error {
	client, err := rpc.Dial("unix", config.Default.Server.Addr)
	if err != nil {
		log.L().Fatal(err.Error())
	}

	rpcArgs := "&server.RPCArgs{}"
	rpcReply := &server.RPCReply{}
	client.Call("RPC.Load", rpcArgs, rpcReply)

	fmt.Printf("Load Plugin!  %v = %v.\n", rpcArgs, rpcReply)
	pluginArgs := "&server.RPCArgs{}"
	pluginReply := &plugins.Reply{}
	client.Call("Service.Load", rpcArgs, pluginReply)

	fmt.Printf("Load Plugin!  %v = %v.\n", pluginArgs, pluginReply)
	return nil
}
