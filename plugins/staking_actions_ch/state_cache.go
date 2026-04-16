package main

import (
	"container/list"

	"github.com/shopspring/decimal"
)

const defaultBucketStateCacheSize = 100000

type cachedBucketState struct {
	info             BucketInfo
	hasInfo          bool
	totalAmount      decimal.Decimal
	nonUnstakeAmount decimal.Decimal
	unstakeCount     int64
}

type pendingBucketState struct {
	latestInfo         *BucketInfo
	latestPositiveInfo *BucketInfo
	totalDelta         decimal.Decimal
	nonUnstakeDelta    decimal.Decimal
	unstakeCountDelta  int64
}

type bucketStateCacheEntry struct {
	bucketID uint64
	state    cachedBucketState
}

type bucketStateCache struct {
	maxEntries int
	ll         *list.List
	cache      map[uint64]*list.Element
}

func newBucketStateCache(maxEntries int) *bucketStateCache {
	if maxEntries <= 0 {
		maxEntries = defaultBucketStateCacheSize
	}
	return &bucketStateCache{
		maxEntries: maxEntries,
		ll:         list.New(),
		cache:      make(map[uint64]*list.Element, maxEntries),
	}
}

func (c *bucketStateCache) Get(bucketID uint64) (cachedBucketState, bool) {
	if c == nil {
		return cachedBucketState{}, false
	}
	if ele, ok := c.cache[bucketID]; ok {
		c.ll.MoveToFront(ele)
		return ele.Value.(*bucketStateCacheEntry).state, true
	}
	return cachedBucketState{}, false
}

func (c *bucketStateCache) Set(bucketID uint64, state cachedBucketState) {
	if c == nil {
		return
	}
	if ele, ok := c.cache[bucketID]; ok {
		c.ll.MoveToFront(ele)
		ele.Value.(*bucketStateCacheEntry).state = state
		return
	}
	ele := c.ll.PushFront(&bucketStateCacheEntry{
		bucketID: bucketID,
		state:    state,
	})
	c.cache[bucketID] = ele
	if c.ll.Len() > c.maxEntries {
		c.removeOldest()
	}
}

func (c *bucketStateCache) removeOldest() {
	if c == nil {
		return
	}
	ele := c.ll.Back()
	if ele == nil {
		return
	}
	c.ll.Remove(ele)
	entry := ele.Value.(*bucketStateCacheEntry)
	delete(c.cache, entry.bucketID)
}
