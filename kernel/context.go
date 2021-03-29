package kernel

import (
	"context"

	"github.com/iotexproject/iotex-core/blockchain/blockdao"
)

type (
	blockDAOCtxKey struct{}
)

// WithBlockDAOCtx add blockDAOCtx into context.
func WithBlockDAOCtx(ctx context.Context, dao blockdao.BlockDAO) context.Context {
	return context.WithValue(ctx, blockDAOCtxKey{}, dao)
}

// GetBlockCtx gets blockDAOCtx
func GetBlockDAOCtx(ctx context.Context) (blockdao.BlockDAO, bool) {
	dao, ok := ctx.Value(blockDAOCtxKey{}).(blockdao.BlockDAO)
	return dao, ok
}
