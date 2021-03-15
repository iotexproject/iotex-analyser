package plugin

import (
	"fmt"
	"net/rpc"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/plugins"
	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"
)

var UnLoad = &cli.Command{
	Name:        "unload",
	Usage:       "unload <PATH>",
	Description: `unload plugin by path`,
	Action:      runUnLoad,
}

func runUnLoad(c *cli.Context) error {
	soPath := c.Args().First()
	if soPath == "" {
		return errors.New("missing <PATH>")
	}
	client, err := rpc.Dial("unix", config.Default.Server.Addr)
	if err != nil {
		return err
	}

	pluginArgs := &plugins.Args{
		Path: soPath,
	}
	pluginReply := &plugins.Reply{}
	if err := client.Call("Service.UnLoad", pluginArgs, pluginReply); err != nil {
		return errors.Wrap(err, "failed to unload plugin")
	}

	fmt.Printf("UnLoad Plugin!  %s.\n", soPath)
	return nil
}
