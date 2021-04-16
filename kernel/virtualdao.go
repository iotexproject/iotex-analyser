package kernel

import (
	"context"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/iotexproject/iotex-core/blockchain/blockdao"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/patrickmn/go-cache"
	"github.com/pkg/errors"
)

var (
	ErrBlockNotExist = errors.New("block not exist")
)

type virtualDAO struct {
	tipHeight uint64
	store     *cache.Cache
}

func NewVirtualDao() blockdao.BlockDAO {
	return &virtualDAO{
		tipHeight: 0,
		store:     cache.New(5*time.Minute, 10*time.Minute),
	}
}

func (vd *virtualDAO) Start(ctx context.Context) error {
	return nil
}

func (vd *virtualDAO) Stop(ctx context.Context) error {
	return nil
}

func (vd *virtualDAO) Height() (uint64, error) {
	return atomic.LoadUint64(&vd.tipHeight), nil
}

func (vd *virtualDAO) GetBlockHash(height uint64) (hash.Hash256, error) {
	return hash.ZeroHash256, nil
}

func (vd *virtualDAO) GetBlockHeight(hash hash.Hash256) (uint64, error) {
	return 0, nil
}

func (vd *virtualDAO) GetBlock(hash hash.Hash256) (*block.Block, error) {
	return nil, nil
}

func (vd *virtualDAO) GetBlockByHeight(height uint64) (*block.Block, error) {
	cacheKey := strconv.FormatUint(height, 10)
	v, ok := vd.store.Get(cacheKey)
	if !ok {
		return nil, ErrBlockNotExist
	}
	return v.(*block.Block), nil
}

func (vd *virtualDAO) Header(hash hash.Hash256) (*block.Header, error) {
	return nil, nil
}

func (vd *virtualDAO) HeaderByHeight(height uint64) (*block.Header, error) {
	return nil, nil
}

func (vd *virtualDAO) FooterByHeight(height uint64) (*block.Footer, error) {
	return nil, nil
}

func (vd *virtualDAO) GetReceipts(height uint64) ([]*action.Receipt, error) {
	cacheKey := strconv.FormatUint(height, 10)
	v, ok := vd.store.Get(cacheKey)
	if !ok {
		return nil, ErrBlockNotExist
	}
	blk := v.(*block.Block)
	return blk.Receipts, nil
}

func (vd *virtualDAO) ContainsTransactionLog() bool {
	return false
}

func (vd *virtualDAO) TransactionLogs(height uint64) (*iotextypes.TransactionLogs, error) {
	cacheKey := strconv.FormatUint(height, 10)
	v, ok := vd.store.Get(cacheKey)
	if !ok {
		return nil, ErrBlockNotExist
	}
	blk := v.(*block.Block)
	log := blk.TransactionLog()
	if log == nil {
		return &iotextypes.TransactionLogs{}, nil
	}
	//memory leak here
	return block.DeserializeSystemLogPb(log.Serialize())
}

func (vd *virtualDAO) PutBlock(ctx context.Context, blk *block.Block) error {
	blkHeight := blk.Height()
	cacheKey := strconv.FormatUint(blkHeight, 10)
	vd.store.Set(cacheKey, blk, cache.DefaultExpiration)
	atomic.StoreUint64(&vd.tipHeight, blkHeight)
	return nil
}

func (vd *virtualDAO) DeleteBlockToTarget(uint64) error {
	return nil
}

func (vd *virtualDAO) GetActionByActionHash(hash.Hash256, uint64) (action.SealedEnvelope, error) {
	return action.SealedEnvelope{}, nil
}

func (vd *virtualDAO) DeleteTipBlock() error {
	return nil
}

func (vd *virtualDAO) GetReceiptByActionHash(hash.Hash256, uint64) (*action.Receipt, error) {
	return nil, nil
}
