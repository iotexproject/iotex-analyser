package tools

import (
	"fmt"
	"net/rpc"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/server"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

var TraceBlock = &cli.Command{
	Name:        "traceblock",
	Usage:       "traceblock --block <height>",
	Description: `trace block in blockDAO`,
	Flags: []cli.Flag{
		&cli.Uint64Flag{
			Name:  "block",
			Usage: "block height",
			Value: 1,
		},
	},
	Action: traceBlock,
}

func traceBlock(c *cli.Context) error {
	client, err := rpc.Dial("unix", config.Default.Server.Addr)
	if err != nil {
		return errors.Wrap(err, "failed to connect RPC server")
	}

	pluginArgs := &server.Args{BlockHeight: c.Uint64("block")}
	pluginReply := &server.Reply{}
	if err := client.Call("Service.TraceBlockByHeight", pluginArgs, pluginReply); err != nil {
		return errors.Wrap(err, "failed to load plugin")
	}

	fmt.Println(pluginReply.Message)
	return nil
}
