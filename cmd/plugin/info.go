package plugin

import (
	"fmt"
	"net/rpc"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/plugins"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

var Info = &cli.Command{
	Name:        "info",
	Usage:       "info",
	Description: `loaded plugin infomation`,
	Action:      runInfo,
}

func runInfo(c *cli.Context) error {

	client, err := rpc.Dial("unix", config.Default.Server.Addr)
	if err != nil {
		return errors.Wrap(err, "failed to connect RPC server")
	}

	pluginArgs := &plugins.Args{}
	pluginReply := &plugins.Reply{}
	if err := client.Call("Service.Info", pluginArgs, pluginReply); err != nil {
		return errors.Wrap(err, "failed to load plugin")
	}

	fmt.Println(pluginReply.Message)
	return nil
}
