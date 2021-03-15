package main

import (
	"context"
	"fmt"

	"github.com/iotexproject/iotex-core/blockchain/block"
)

type blockPlugin struct{}

func (b blockPlugin) Name() string {
	return "block"
}
func (b blockPlugin) Start(ctx context.Context) error {
	fmt.Println("block plugin start")
	return nil
}

func (b blockPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	fmt.Println("block plugin putblock")
	return nil
}

func (b blockPlugin) Stop(ctx context.Context) error {
	fmt.Println("block plugin stop")
	return nil
}

// exported
var Plugin blockPlugin
