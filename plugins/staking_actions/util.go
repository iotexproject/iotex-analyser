package main

import (
	"context"
	"math/big"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
)

func getCandidateAddressByName(name string, height uint64) (string, error) {
	candidate := &models.Candidate{}
	if err := candidate.FetchByNameWithHeight(name, height); err != nil {
		return "", errors.Wrap(err, "failed to fetch candidate by name")
	}
	return candidate.CandidateID, nil
}

func bucketSumAmount(ctx context.Context, conn driver.Conn, bucketID string, stash *stash) (decimal.Decimal, error) {
	amount, err := getBucketSumAmountByBucketID(ctx, conn, bucketID)
	if err != nil {
		return decimal.NewFromInt(0), errors.Wrap(err, "failed to get bucket sum amount by bucket id")
	}
	s := stash.sumAmount[bucketID]
	return amount.Add(decimal.NewFromBigInt(&s, 0)), nil
}

func getBucketSumAmountByBucketID(ctx context.Context, conn driver.Conn, bucketID string) (decimal.Decimal, error) {
	amount := big.NewInt(0)
	zero := decimal.NewFromInt(0)
	if err := conn.QueryRow(ctx, "SELECT sum(toUInt256(amount)) FROM ? WHERE bucket_id=?", models.StakingActions{}.TableName(), bucketID).Scan(&amount); err != nil {
		return zero, errors.Wrap(err, "failed to get sum amount by bucket id")
	}
	if amount.Sign() == 0 {
		return zero, nil
	}
	decmailAmount, err := decimal.NewFromString(amount.String())
	if err != nil {
		return zero, errors.Wrap(err, "failed to convert string to decimal")
	}
	return decmailAmount, nil
}

func getFixBucketSumAmountByBucketID(ctx context.Context, tx driver.Conn, bucketID string) (decimal.Decimal, error) {
	amount := big.NewInt(0)
	var count uint64
	zero := decimal.NewFromInt(0)
	err := tx.QueryRow(ctx, "SELECT count(*) FROM ? WHERE bucket_id=? and act_type='Unstake'", models.StakingActions{}.TableName(), bucketID).Scan(&count)
	if err != nil {
		return zero, errors.Wrap(err, "failed to get count by bucket id")
	}
	if count == 0 {
		return zero, nil
	}
	if err := tx.QueryRow(ctx, "SELECT sum(toUInt256(amount)) FROM ? WHERE bucket_id=? and act_type!='Unstake'", models.StakingActions{}.TableName(), bucketID).Scan(&amount); err != nil {
		return zero, errors.Wrap(err, "failed to get sum amount by bucket id")
	}
	if amount.Sign() == 0 {
		return zero, nil
	}
	decmailAmount, err := decimal.NewFromString(amount.String())
	if err != nil {
		return zero, errors.Wrap(err, "failed to convert string to decimal")
	}
	return decmailAmount, nil
}

type BucketInfo struct {
	OwnerAddress string `ch:"owner_address"`
	Candidate    string `ch:"candidate"`
	AutoStake    bool   `ch:"auto_stake"`
	Duration     uint32 `ch:"duration"`
}

func getBucketInfoAddressByBucketID(ctx context.Context, tx driver.Conn, bucketID string) (*BucketInfo, error) {
	var bi BucketInfo
	if err := tx.QueryRow(ctx, "SELECT owner_address,candidate,auto_stake,duration FROM ? WHERE bucket_id=? ORDER BY block_height DESC, log_index DESC LIMIT 1", models.StakingActions{}.TableName(), bucketID).ScanStruct(&bi); err != nil {
		return nil, errors.Wrap(err, "failed to get bucket info by bucket id")
	}
	return &bi, nil
}

func bucketInfo(ctx context.Context, tx driver.Conn, bucketID string, stash *stash) (*BucketInfo, error) {
	if bi, ok := stash.info[bucketID]; ok {
		return bi, nil
	}
	bi, err := getBucketInfoAddressByBucketID(ctx, tx, bucketID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get bucket info by bucket id")
	}
	return bi, nil
}
