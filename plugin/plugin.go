package plugin

import (
	"context"

	"github.com/iotexproject/iotex-core/blockchain/block"
)

type Type int

const (
	TypeStandard Type = iota
	TypeWorker        //independence worker does not implement PutBlock
)

type Adapter interface {
	Name() string
	Version() string
	Type() Type
	Start(context.Context) error
	Stop(context.Context) error
	PutBlock(context.Context, *block.Block) error
}

type BatchAdapter interface {
	Adapter
	PutBlocks(ctx context.Context, blks []*block.Block) error
	BatchSize() int
}

type DependentAdapter interface {
	DependentPlugins() []string
}
