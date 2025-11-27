package plugin

import (
	"context"

	"github.com/iotexproject/iotex-core/v2/blockchain/block"
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

type PluginShadow struct {
	ShadowName  func(string) string
	ShadowTable func(a any) any
}

var (
	PluginSelf = PluginShadow{
		ShadowName:  func(s string) string { return s },
		ShadowTable: func(a any) any { return a },
	}
)
