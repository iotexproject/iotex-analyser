package plugins

import (
	"context"

	"github.com/iotexproject/iotex-core/blockchain/block"
)

type Plugin interface {
	Name() string
	Start(context.Context) error
	Stop(context.Context) error
	PutBlock(context.Context, *block.Block) error
}
