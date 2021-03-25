package kernel

import (
	"context"
	"sync"

	"github.com/iotexproject/go-pkgs/cache"
	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/iotexproject/iotex-core/blockchain/blockdao"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
)

var (
	ErrBlockNotExist = errors.New("block not exist")
)

type virtualDAO struct {
	mu        sync.RWMutex
	tipHeight uint64
	store     *cache.ThreadSafeLruCache
}

func NewVirtualDao() blockdao.BlockDAO {
	return &virtualDAO{
		mu:        sync.RWMutex{},
		tipHeight: 0,
		store:     cache.NewThreadSafeLruCache(32),
	}
}

func (vd *virtualDAO) Start(ctx context.Context) error {
	return nil
}

func (vd *virtualDAO) Stop(ctx context.Context) error {
	return nil
}

func (vd *virtualDAO) Height() (uint64, error) {
	vd.mu.RLock()
	defer vd.mu.RUnlock()
	height := vd.tipHeight
	return height, nil
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
	v, ok := vd.store.Get(height)
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
	v, ok := vd.store.Get(height)
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
	v, ok := vd.store.Get(height)
	if !ok {
		return nil, ErrBlockNotExist
	}
	blk := v.(*block.Block)
	log := blk.TransactionLog()
	if log == nil {
		return &iotextypes.TransactionLogs{}, nil
	}
	return block.DeserializeSystemLogPb(log.Serialize())
}

func (vd *virtualDAO) PutBlock(ctx context.Context, blk *block.Block) error {
	blkHeight := blk.Height()
	vd.store.Add(blkHeight, blk)
	vd.mu.Lock()
	vd.tipHeight = blkHeight
	vd.mu.Unlock()
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
