package main

import (
	"context"
	"encoding/json"
	"math"
	"math/big"
	"time"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-core/blockchain/genesis"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Delegate struct {
	OperatorAddress string
	RewardAddress   string
	OwnerAddress    string
	Candidate       string
	Active          bool
	Name            string
	StakeAmount     *big.Int
	VoteWeight      *big.Int
	SelfStake       bool
	Rank            int
	VoteRate        int
	Productivity    int
}

func delegate() error {
	pluginHeight, err := db.GetIndexHeight("staking_actions")
	if err != nil {
		return errors.WithStack(err)
	}
	epochNumber := kernel.GetEpochNum(pluginHeight)
	request := &iotexapi.GetEpochMetaRequest{EpochNumber: epochNumber}
	chainClient := kernel.ChainClient()
	epochMeta, err := chainClient.GetEpochMeta(context.Background(), request)
	if err != nil {
		return errors.WithStack(err)
	}
	raw, _ := json.Marshal(epochMeta.GetBlockProducersInfo())
	store := &db.Store{
		Key:   "current_block_producer_info",
		Value: string(raw),
	}
	if err := store.Save(); err != nil {
		return errors.WithStack(err)
	}

	stakings, err := getCandidateStaking(pluginHeight)
	if err != nil {
		return errors.WithStack(err)
	}
	delegateActives := getDelegateActive(pluginHeight)
	candidates, err := models.GetAllCandidates()
	if err != nil {
		return errors.WithStack(err)
	}
	delegateMap, err := getDelegateMap(epochNumber, stakings, candidates, delegateActives)
	if err != nil {
		return errors.WithStack(err)
	}
	probationList := getProbationList(pluginHeight)
	err = db.DB().Transaction(func(tx *gorm.DB) error {
		tx.Where("1 = 1").Delete(&models.Delegate{})
		for c, d := range delegateMap {
			probated := false
			if _, ok := probationList[d.OperatorAddress]; ok {
				probated = true
			}
			modelDelegate := models.Delegate{
				BlockHeight:     pluginHeight,
				OperatorAddress: d.OperatorAddress,
				RewardAddress:   d.RewardAddress,
				OwnerAddress:    d.OwnerAddress,
				Candidate:       c,
				Active:          d.Active,
				Name:            d.Name,
				StakeAmount:     decimal.NewFromBigInt(d.StakeAmount, 0),
				VoteWeight:      decimal.NewFromBigInt(d.VoteWeight, 0),
				SelfStake:       d.SelfStake,
				Productivity:    d.Productivity,
				Probated:        probated,
			}
			if err := tx.Create(&modelDelegate).Error; err != nil {
				return err
			}
		}
		return nil
	})
	return err
}

func getDelegateMap(epochNumber uint64, stakings []*Staking, candidates models.Candidates, delegateActives map[string]int) (map[string]*Delegate, error) {
	delegateMap := make(map[string]*Delegate)
	totalVotes := big.NewInt(0)
	for _, staking := range stakings {
		delegate, ok := delegateMap[staking.Candidate]
		if !ok {
			cand, err := candidates.ByOwnerAddress(staking.Candidate)
			if err != nil {
				return delegateMap, err
			}
			active := false
			productionNum := 0
			//cand.OperatorAddress is block producer address
			if productivity, ok := delegateActives[cand.OperatorAddress]; ok {
				active = true
				productionNum = productivity
			}
			delegate = &Delegate{
				Name:            cand.Name,
				OwnerAddress:    staking.OwnerAddress,
				Candidate:       staking.Candidate,
				OperatorAddress: cand.OperatorAddress,
				RewardAddress:   cand.RewardAddress,
				Active:          active,
				StakeAmount:     big.NewInt(0),
				VoteWeight:      big.NewInt(0),
				SelfStake:       isSelfStake(staking.Candidate, epochNumber),
				Productivity:    productionNum,
			}
		}
		stakeAmount, _ := big.NewInt(0).SetString(staking.Amount, 0)
		delegate.StakeAmount = delegate.StakeAmount.Add(delegate.StakeAmount, stakeAmount)
		voteBucket := &VoteBucket{
			StakedAmount:   stakeAmount,
			AutoStake:      staking.AutoStake,
			StakedDuration: staking.Duration,
		}
		selfAutoStake := false
		if staking.OwnerAddress == staking.Candidate {
			selfAutoStake = true
		}
		voteWeight := calculateVoteWeight(config.Default.Genesis.Staking.VoteWeightCalConsts, voteBucket, selfAutoStake)
		delegate.VoteWeight = delegate.VoteWeight.Add(delegate.VoteWeight, voteWeight)
		totalVotes = totalVotes.Add(totalVotes, voteWeight)
		delegateMap[staking.Candidate] = delegate
	}
	return delegateMap, nil
}

type Staking struct {
	ID           uint64
	BlockHeight  uint64
	BucketID     uint64
	OwnerAddress string
	Candidate    string
	Amount       string
	ActType      string
	AutoStake    bool
	Duration     uint32
}
type VoteBucket struct {
	Index            uint64
	Candidate        string
	Owner            string
	StakedAmount     *big.Int
	StakedDuration   uint32
	CreateTime       time.Time
	StakeStartTime   time.Time
	UnstakeStartTime time.Time
	AutoStake        bool
}

func calculateVoteWeight(c genesis.VoteWeightCalConsts, v *VoteBucket, selfStake bool) *big.Int {
	remainingTime := float64(v.StakedDuration * 86400)
	weight := float64(1)
	var m float64
	if v.AutoStake {
		m = c.AutoStake
	}
	if remainingTime > 0 {
		weight += math.Log(math.Ceil(remainingTime/86400)*(1+m)) / math.Log(c.DurationLg) / 100
	}
	if selfStake && v.AutoStake && v.StakedDuration >= 91 {
		// self-stake extra bonus requires enable auto-stake for at least 3 months
		weight *= c.SelfStake
	}

	amount := new(big.Float).SetInt(v.StakedAmount)
	weightedAmount, _ := amount.Mul(amount, big.NewFloat(weight)).Int(nil)
	return weightedAmount
}

func getCandidateStaking(height uint64) ([]*Staking, error) {
	db := db.DB()
	query := "select id,block_height,bucket_id,owner_address,candidate,(select sum(b.amount) from staking_actions b where b.block_height<=? and b.bucket_id=a.bucket_id) as amount,act_type,auto_stake,duration from staking_actions a where id=any(array(select max(id) from staking_actions where block_height<=? group by bucket_id))"
	rows, err := db.Raw(query, height, height).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []*Staking
	for rows.Next() {
		av := new(Staking)

		if err := db.ScanRows(rows, av); err != nil {
			return nil, err
		}
		results = append(results, av)
	}
	return results, nil
}

// check candidate register and amount >= 1200000000000000000000000
// only need check hermes_voting_results , epoch_number=1 staking_address=?
func isSelfStake(candidate string, epochNumber uint64) bool {
	var count int64
	if err := db.DB().Model(&models.HermesVotingResult{}).Where("epoch_number=? and staking_address=? and self_staking>=1200000000000000000000000", epochNumber, candidate).Count(&count).Error; err != nil {
		return false
	}
	if count > 0 {
		return true
	}
	return false
}

func getDelegateActive(height uint64) map[string]int {
	epochNumber := kernel.GetEpochNum(height)
	delegateActive := make(map[string]int)
	startHeight := kernel.GetEpochHeight(epochNumber)
	db := db.DB()
	var actives []struct {
		ProducerAddress string
		Active          int
	}
	if err := db.Model(&models.Block{}).Select("producer_address, count(*) as active").Where("block_height >= ? and block_height <= ?", startHeight, height).Group("producer_address").Find(&actives).Error; err != nil {
		return delegateActive
	}
	for _, a := range actives {
		delegateActive[a.ProducerAddress] = a.Active
	}
	return delegateActive
}

func getProbationList(height uint64) map[string]struct{} {
	epochNumber := kernel.GetEpochNum(height)
	probationList := make(map[string]struct{})
	var addresses []struct {
		Address string
	}
	db := db.DB()
	if err := db.Model(&models.Probation{}).Select("address").Where("epoch_number= ?", epochNumber).Scan(&addresses).Error; err != nil {
		return probationList
	}
	for _, a := range addresses {
		probationList[a.Address] = struct{}{}
	}
	return probationList
}
