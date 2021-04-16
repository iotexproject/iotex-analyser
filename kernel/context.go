package kernel

import (
	"context"

	"github.com/iotexproject/iotex-core/blockchain/blockdao"
)

type (
	blockDAOCtxKey struct{}
	configCtxKey   struct{}
)

// WithConfigCtx add config path into context.
func WithConfigCtx(ctx context.Context, config string) context.Context {
	return context.WithValue(ctx, configCtxKey{}, config)
}

// GetConfigCtx gets configCtx
func GetConfigCtx(ctx context.Context) (string, bool) {
	c, ok := ctx.Value(configCtxKey{}).(string)
	return c, ok
}

// WithBlockDAOCtx add blockDAOCtx into context.
func WithBlockDAOCtx(ctx context.Context, dao blockdao.BlockDAO) context.Context {
	return context.WithValue(ctx, blockDAOCtxKey{}, dao)
}

// GetBlockDAOCtx gets blockDAOCtx
func GetBlockDAOCtx(ctx context.Context) (blockdao.BlockDAO, bool) {
	dao, ok := ctx.Value(blockDAOCtxKey{}).(blockdao.BlockDAO)
	return dao, ok
}
