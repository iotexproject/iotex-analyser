package models

import (
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

type VoteBucketList struct {
	EpochNumber uint64 `gorm:"primary_key;" sql:"type:bigint"`
	BucketList  []byte
}

func (VoteBucketList) TableName() string {
	return "vote_bucketlist"
}

func GetVoteBucketList(epochNum uint64) (*iotextypes.VoteBucketList, error) {
	voteBucketListAll := &iotextypes.VoteBucketList{}
	var vbl VoteBucketList
	if err := db.DB().Where("epoch_number = ?", epochNum).First(&vbl).Error; err != nil {
		return nil, errors.Wrapf(err, "failed to get vote bucket list in epoch %d", epochNum)
	}
	if err := proto.Unmarshal(vbl.BucketList, voteBucketListAll); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal vote bucket list in epoch %d", epochNum)
	}
	return voteBucketListAll, nil
}
