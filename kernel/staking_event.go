package kernel

import (
	"bytes"
	"fmt"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/pkg/util/byteutil"
	"github.com/pkg/errors"
)

type CandidateRegisterEvent struct {
	BucketID  uint64
	Candidate address.Address
}

type CandidateActivateEvent struct {
	BucketID  uint64
	Candidate address.Address
}

type CandidateEndorsementEvent struct {
	BucketID  uint64
	Candidate address.Address
	Op        action.CandidateEndorsementOp
}

var (
	stakingProtocolHash                = hash.Hash160b([]byte("staking"))
	stakingProtocolAddress, _          = address.FromBytes(stakingProtocolHash[:])
	candidateRegisterTopicStr          = "candidateRegister"
	candidateActivateTopicStr          = "candidateActivate"
	candidateEndorsementTopicStr       = "candidateEndorsement"
	candidateEndorsementWithOpTopicStr = "candidateEndorsementWithOp"
)

// ParseCandidateRegisterEvent parses candidate register event from logs
func ParseCandidateRegisterEvent(logs []*action.Log, postFairbankMigration bool, postBLS bool) (*CandidateRegisterEvent, error) {
	fmt.Printf("parse candidate register event, postFairbankMigration: %v, postBLS: %v\n", postFairbankMigration, postBLS)
	var event *CandidateRegisterEvent
	var err error
	for _, log := range logs {
		if !postFairbankMigration {
			event, err = parseCandidateRegisterLogPreFairbankMigration(log)
		} else if !postBLS {
			event, err = parseCandidateRegisterLogPostFairbankMigration(log, postBLS)
		} else {
			event, err = parseCandidateRegisterLogPostBLS(log)
		}
		if err != nil {
			return nil, err
		}
		if event != nil {
			return event, nil
		}
	}
	return nil, nil
}

// ParseCandidateActivateEvent parses candidate activate event from logs
func ParseCandidateActivateEvent(logs []*action.Log) (*CandidateActivateEvent, error) {
	for _, log := range logs {
		if log.Address != stakingProtocolAddress.String() {
			continue
		}
		topic := hash.BytesToHash256([]byte(candidateActivateTopicStr))
		if !bytes.Equal(log.Topics[0][:], topic[:]) {
			continue
		}
		bucketIdx := byteutil.BytesToUint64BigEndian(log.Topics[1][24:])
		cand, err := address.FromBytes(log.Topics[2][:])
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse candidate address from log topics")
		}
		event := &CandidateActivateEvent{
			BucketID:  bucketIdx,
			Candidate: cand,
		}
		return event, nil
	}
	return nil, nil
}

// ParseCandidateEndorsementEvent parses candidate endorsement event from logs
func ParseCandidateEndorsementEvent(logs []*action.Log) (*CandidateEndorsementEvent, error) {
	topic := hash.BytesToHash256([]byte(candidateEndorsementTopicStr))
	topicWithOp := hash.BytesToHash256([]byte(candidateEndorsementWithOpTopicStr))
	var event *CandidateEndorsementEvent
	for _, log := range logs {
		if log.Address != stakingProtocolAddress.String() {
			continue
		}
		if bytes.Equal(log.Topics[0][:], topic[:]) {
			bucketIdx := byteutil.BytesToUint64BigEndian(log.Topics[1][24:])
			cand, err := address.FromBytes(log.Topics[2][:])
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse candidate address from log topics")
			}
			op := action.CandidateEndorsementOpEndorse
			if log.Topics[3][31] == 0 {
				op = action.CandidateEndorsementOpIntentToRevoke
			}
			event = &CandidateEndorsementEvent{
				BucketID:  bucketIdx,
				Candidate: cand,
				Op:        op,
			}
			return event, nil
		} else if bytes.Equal(log.Topics[0][:], topicWithOp[:]) {
			bucketIdx := byteutil.BytesToUint64BigEndian(log.Topics[1][24:])
			cand, err := address.FromBytes(log.Topics[2][:])
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse candidate address from log topics")
			}
			op := action.CandidateEndorsementOp(log.Topics[3][31])
			event = &CandidateEndorsementEvent{
				BucketID:  bucketIdx,
				Candidate: cand,
				Op:        op,
			}
			return event, nil
		}
	}
	return nil, nil
}

func parseCandidateRegisterLogPreFairbankMigration(log *action.Log) (*CandidateRegisterEvent, error) {
	if log.Address != stakingProtocolAddress.String() {
		return nil, nil
	}
	topic := hash.Hash256b([]byte(candidateRegisterTopicStr))
	if !bytes.Equal(log.Topics[0][:], topic[:]) {
		return nil, nil
	}
	bucketIdx := byteutil.BytesToUint64BigEndian(log.Data)
	return &CandidateRegisterEvent{
		BucketID: bucketIdx,
	}, nil
}

func parseCandidateRegisterLogPostFairbankMigration(log *action.Log, postBLS bool) (*CandidateRegisterEvent, error) {
	if log.Address != stakingProtocolAddress.String() {
		return nil, nil
	}
	topic := hash.BytesToHash256([]byte(candidateRegisterTopicStr))
	if !bytes.Equal(log.Topics[0][:], topic[:]) {
		return nil, nil
	}
	bucketIdx := byteutil.BytesToUint64BigEndian(log.Topics[1][24:])
	cand, err := address.FromBytes(log.Topics[2][:])
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse candidate address from log topics")
	}
	return &CandidateRegisterEvent{
		BucketID:  bucketIdx,
		Candidate: cand,
	}, nil
}

func parseCandidateRegisterLogPostBLS(log *action.Log) (*CandidateRegisterEvent, error) {
	if log.Address != stakingProtocolAddress.String() {
		return nil, nil
	}
	stk := action.NativeStakingContractABI()
	event := stk.Events["CandidateActivated"]
	if !bytes.Equal(event.ID.Bytes(), log.Topics[0][:]) {
		return nil, nil
	}
	cand, err := address.FromBytes(log.Topics[1][:])
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse candidate address from log topics")
	}
	paramsNonIndexed, err := event.Inputs.Unpack(log.Data)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unpack event inputs from log data")
	}
	bucketID := paramsNonIndexed[0].(uint64)
	return &CandidateRegisterEvent{
		BucketID:  bucketID,
		Candidate: cand,
	}, nil
}
