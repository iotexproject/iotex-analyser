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

// CatchUpAdapter is implemented by plugins that opt in to running in
// catch-up mode (where the index starts mid-chain at the current tip
// instead of from height 0).
//
// Plugins that maintain cumulative state derived from full history —
// balances, holders, lifetime totals, etc. — should NOT implement this,
// or should return false: starting from an arbitrary height would
// produce permanently incorrect data. Per-block fact plugins (block
// metadata, receipts, action records) and snapshot-derivable plugins
// (state queried from chain at each height) are safe candidates.
//
// Operators can override the safety check via iotex.catchUpAllowPlugins
// in config.
type CatchUpAdapter interface {
	Adapter
	CatchUpSafe() bool
}

type PluginShadow struct {
	ShadowName  func(string) string
	ShadowTable func(a Table) Table
}

type Table interface {
	TableName() string
}

var (
	PluginSelf = PluginShadow{
		ShadowName:  func(s string) string { return s },
		ShadowTable: func(a Table) Table { return a },
	}
)
