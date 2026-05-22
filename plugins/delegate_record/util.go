package main

import (
	"context"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-core/v2/blockchain/genesis"
	"github.com/iotexproject/iotex-core/v2/pkg/log"
	"github.com/iotexproject/iotex-core/v2/state"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
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
	Productivity    int
}

func delegate(ctx context.Context) error {
	log.L().Debug("delegate record start")
	blockTime := time.Now().UTC()
	pluginHeight, err := db.GetIndexHeight("staking_actions")
	if err != nil {
		return errors.WithStack(err)
	}
	// SelfStake is resolved from candidate_self_stake; only run once that plugin
	// has indexed at least as far as staking_actions, so isSelfStake never
	// queries beyond the data it has populated.
	cssHeight, err := db.GetIndexHeight("candidate_self_stake")
	if err != nil || cssHeight < pluginHeight {
		log.L().Warn("candidate_self_stake behind staking_actions, skipping delegate_record run",
			zap.Uint64("css_height", cssHeight), zap.Uint64("plugin_height", pluginHeight), zap.Error(err))
		return nil
	}
	epochNumber := kernel.GetEpochNum(pluginHeight)
	blk, err := kernel.GetBlockByHeightFromChain(ctx, pluginHeight)
	if err == nil {
		blockTime = time.Unix(blk.Timestamp().Unix(), 0)
	}

	// request := &iotexapi.GetEpochMetaRequest{EpochNumber: epochNumber}
	// chainClient := kernel.ChainClient()
	// epochMeta, err := chainClient.GetEpochMeta(context.Background(), request)
	// if err != nil {
	// 	return errors.WithStack(err)
	// }
	// raw, _ := json.Marshal(epochMeta.GetBlockProducersInfo())
	// store := &db.Store{
	// 	Key:   "current_block_producer_info_delegate_record",
	// 	Value: string(raw),
	// }
	// if err := store.Save(); err != nil {
	// 	return errors.WithStack(err)
	// }
	err = makeMaxIDTempTable(pluginHeight)
	if err != nil {
		return errors.WithStack(err)
	}
	bucketSumAmount, err := getBucketSumAmount(pluginHeight)
	if err != nil {
		return errors.WithStack(err)
	}
	stakings, err := getCandidateStaking(pluginHeight)
	if err != nil {
		return errors.WithStack(err)
	}
	systemStakings, err := getSystemStaking(pluginHeight)
	if err != nil {
		return errors.WithStack(err)
	}
	systemStakingsV2, err := getSystemStakingV2(pluginHeight)
	if err != nil {
		return errors.WithStack(err)
	}
	delegateActives := getDelegateActive(pluginHeight)
	candidates, err := models.GetAllCandidates()
	if err != nil {
		return errors.WithStack(err)
	}
	delegateMap, err := getDelegateMap(epochNumber, pluginHeight, stakings, systemStakings, systemStakingsV2, candidates, delegateActives, bucketSumAmount)
	if err != nil {
		return errors.WithStack(err)
	}
	if len(delegateMap) == 0 {
		return nil
	}
	probationList := getProbationList(pluginHeight)
	err = db.DB().Transaction(func(tx *gorm.DB) error {
		for c, d := range delegateMap {
			probated := false
			if _, ok := probationList[d.OperatorAddress]; ok {
				probated = true
			}
			modelDelegate := models.DelegateRecord{
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
				Timestamp:       blockTime,
			}
			if err := tx.Create(&modelDelegate).Error; err != nil {
				return err
			}
		}
		return db.UpdateIndexHeightByTx(tx, "delegate_record", pluginHeight)
	})
	return err
}

func getDelegateMap(epochNumber, pluginHeight uint64, stakings []*Staking, systemStakings, systemStakingsV2 []*SystemStakingBucket, candidates models.Candidates, delegateActives map[string]int, bucketAmount map[uint64]*big.Int) (map[string]*Delegate, error) {
	delegateMap := make(map[string]*Delegate)
	totalVotes := big.NewInt(0)
	for _, cand := range getNotStakeCandidateList(candidates, stakings) {
		if _, ok := delegateMap[cand.CandidateID]; !ok {
			active := false
			productionNum := 0
			//cand.OperatorAddress is block producer address
			if productivity, ok := delegateActives[cand.OperatorAddress]; ok {
				active = true
				productionNum = productivity
			}
			delegate := &Delegate{
				Name:            cand.Name,
				OwnerAddress:    cand.OwnerAddress,
				Candidate:       cand.CandidateID,
				OperatorAddress: cand.OperatorAddress,
				RewardAddress:   cand.RewardAddress,
				Active:          active,
				StakeAmount:     big.NewInt(0),
				VoteWeight:      big.NewInt(0),
				SelfStake:       false,
				Productivity:    productionNum,
			}
			delegateMap[cand.CandidateID] = delegate
		}
	}
	for _, staking := range stakings {
		delegate, ok := delegateMap[staking.Candidate]
		if !ok {
			cand, err := candidates.ByCandidateID(staking.Candidate)
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
				OwnerAddress:    cand.OwnerAddress,
				Candidate:       staking.Candidate,
				OperatorAddress: cand.OperatorAddress,
				RewardAddress:   cand.RewardAddress,
				Active:          active,
				StakeAmount:     big.NewInt(0),
				VoteWeight:      big.NewInt(0),
				SelfStake:       isSelfStake(staking.Candidate, pluginHeight),
				Productivity:    productionNum,
			}
		}

		stakeAmount, ok := bucketAmount[staking.BucketID]
		if !ok {
			return delegateMap, errors.Errorf("can not found bucketAmount with bucketID: %d", staking.BucketID)
		}
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
	for _, staking := range systemStakings {
		delegate, ok := delegateMap[staking.DelegateOwnerAddress]
		if !ok {
			continue
		}
		stakeAmount, _ := big.NewInt(0).SetString(staking.StakedAmount, 0)
		delegate.StakeAmount = delegate.StakeAmount.Add(delegate.StakeAmount, stakeAmount)
		voteWeight, _ := big.NewInt(0).SetString(staking.VotingPower, 0)
		delegate.VoteWeight = delegate.VoteWeight.Add(delegate.VoteWeight, voteWeight)
		totalVotes = totalVotes.Add(totalVotes, voteWeight)
		delegateMap[staking.DelegateOwnerAddress] = delegate
	}
	for _, staking := range systemStakingsV2 {
		delegate, ok := delegateMap[staking.DelegateOwnerAddress]
		if !ok {
			continue
		}
		stakeAmount, _ := big.NewInt(0).SetString(staking.StakedAmount, 0)
		delegate.StakeAmount = delegate.StakeAmount.Add(delegate.StakeAmount, stakeAmount)
		voteWeight, _ := big.NewInt(0).SetString(staking.VotingPower, 0)
		delegate.VoteWeight = delegate.VoteWeight.Add(delegate.VoteWeight, voteWeight)
		totalVotes = totalVotes.Add(totalVotes, voteWeight)
		delegateMap[staking.DelegateOwnerAddress] = delegate
	}

	candidateListv2, err := models.GetCandidateList(epochNumber)
	if err != nil {
		return nil, err
	}
	for _, cand := range candidateListv2.GetCandidates() {
		votes, ok := big.NewInt(0).SetString(cand.TotalWeightedVotes, 10)
		if !ok {
			continue
		}
		for _, d := range delegateMap {
			if strings.EqualFold(cand.OperatorAddress, d.OperatorAddress) {
				d.VoteWeight = votes
				break
			}
		}
	}
	candidateList, err := GetProducerCandidateList(kernel.ChainClient(), epochNumber)
	if err == nil {
		//currently some delegate votes is not correct, we directly use chainMeta data
		for _, cand := range candidateList {
			for _, d := range delegateMap {
				if strings.EqualFold(cand.Address, d.OperatorAddress) {
					d.VoteWeight = cand.Votes
					break
				}
			}
		}
	}
	return delegateMap, nil
}

func getNotStakeCandidateList(candidates models.Candidates, stakings []*Staking) models.Candidates {
	stakingsMap := make(map[string]struct{})
	for _, stakeing := range stakings {
		stakingsMap[stakeing.Candidate] = struct{}{}
	}

	var cs models.Candidates
	for _, candidate := range candidates {
		if _, found := stakingsMap[candidate.CandidateID]; !found {
			cs = append(cs, candidate)
		}
	}

	return cs
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
type SystemStakingBucket struct {
	ID                   uint64
	BucketID             uint64
	BlockHeight          uint64
	CreateTime           int64
	StakeStartTime       int64
	UnstakeStartTime     int64
	StakedAmount         string
	VotingPower          string
	OwnerAddress         string
	DelegateOwnerAddress string
	Amount               string
	EventType            string
	Sender               string
	Recipient            string
	ActHash              string
	Timestamp            int64
	AutoStake            bool
	Duration             uint32
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

func makeMaxIDTempTable(blockHeight uint64) error {
	query := "CREATE TABLE  IF NOT EXISTS temp_staking_id_record (id bigint PRIMARY KEY)"
	if err := db.DB().Exec(query).Error; err != nil {
		return err
	}
	//empty temp table
	if err := db.DB().Exec("truncate table temp_staking_id_record").Error; err != nil {
		return err
	}
	query = "insert into temp_staking_id_record select max(id) from staking_actions where block_height<=? group by bucket_id"
	if err := db.DB().Exec(query, blockHeight).Error; err != nil {
		return err
	}
	return nil
}

func getBucketSumAmount(height uint64) (map[uint64]*big.Int, error) {
	db := db.DB()
	query := "select bucket_id,sum(amount) as amount from staking_actions where block_height<=? group by bucket_id"
	rows, err := db.Raw(query, height).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results = make(map[uint64]*big.Int)
	for rows.Next() {
		var bucketID uint64
		var amount string
		if err := rows.Scan(&bucketID, &amount); err != nil {
			return nil, err
		}
		amountInt, _ := big.NewInt(0).SetString(amount, 0)
		results[bucketID] = amountInt
	}
	return results, nil
}

func getCandidateStaking(height uint64) ([]*Staking, error) {
	db := db.DB()
	query := "SELECT a.id, a.block_height, a.bucket_id, a.owner_address, a.candidate, a.amount, a.act_type, a.auto_stake, a.duration FROM staking_actions a INNER JOIN temp_staking_id_record b ON a.id = b.id"
	rows, err := db.Raw(query).Rows()
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

func getSystemStaking(height uint64) ([]*SystemStakingBucket, error) {
	db := db.DB()
	query := "select t1.* from system_staking_buckets t1 INNER JOIN (select MAX(id)AS max_id  from system_staking_buckets t4 where block_height<=? GROUP BY bucket_id) as t2 on t2.max_id=t1.id"
	rows, err := db.Raw(query, height).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []*SystemStakingBucket
	for rows.Next() {
		av := new(SystemStakingBucket)
		if err := db.ScanRows(rows, av); err != nil {
			return nil, err
		}
		results = append(results, av)
	}
	return results, nil
}

func getSystemStakingV2(height uint64) ([]*SystemStakingBucket, error) {
	db := db.DB()
	query := "select t1.* from system_staking_buckets_v2 t1 INNER JOIN (select MAX(id)AS max_id  from system_staking_buckets_v2 t4 where block_height<=? GROUP BY bucket_id) as t2 on t2.max_id=t1.id"
	rows, err := db.Raw(query, height).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var results []*SystemStakingBucket
	for rows.Next() {
		av := new(SystemStakingBucket)
		if err := db.ScanRows(rows, av); err != nil {
			return nil, err
		}
		results = append(results, av)
	}
	return results, nil
}

// isSelfStake reports whether the candidate holds a self-stake bucket as of the
// given block height.
//
// Before the Xingu hardfork a self-stake bucket was immutable at >= 1.2M IOTX,
// so "self_staking >= 1.2M" in hermes_voting_results was a valid proxy for
// having one. Xingu introduced unproductive-delegate slashing, which draws down
// the self-stake bucket and can push a still-valid bucket below 1.2M. We
// therefore look the bucket up directly: candidate_self_stake records the
// latest self-stake bucket per candidate, and bucket_id == MaxUint64 means the
// candidate currently has none.
func isSelfStake(candidate string, height uint64) bool {
	record := &models.CandidateSelfStake{}
	if err := record.FetchByCandidateIDWithHeight(candidate, height, db.DB()); err != nil {
		if err == gorm.ErrRecordNotFound {
			return false
		}
		log.L().Error("failed to query candidate self stake", zap.String("candidate", candidate), zap.Error(err))
		return false
	}
	return record.BucketID != math.MaxUint64
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

func getAllDelegateByHeight(height uint64) map[string]struct{} {
	delegates := make(map[string]struct{})
	var addresses []struct {
		OwnerAddress string
	}
	db := db.DB()
	if err := db.Model(&models.DelegateRecord{}).Select("owner_address").Where("block_height = ?", height).Scan(&addresses).Error; err != nil {
		return delegates
	}
	for _, a := range addresses {
		delegates[a.OwnerAddress] = struct{}{}
	}
	return delegates
}

func GetProducerCandidateList(cli iotexapi.APIServiceClient, epochNum uint64) (state.CandidateList, error) {
	request := &iotexapi.ReadStateRequest{
		ProtocolID: []byte("poll"),
		MethodName: []byte("BlockProducersByEpoch"),
		Arguments:  [][]byte{[]byte(strconv.FormatUint(epochNum, 10))},
	}
	bpResponse, err := cli.ReadState(context.Background(), request)
	if err != nil {
		return nil, err
	}
	var BPs state.CandidateList
	if err := BPs.Deserialize(bpResponse.Data); err != nil {
		return nil, err
	}
	return BPs, nil
}
