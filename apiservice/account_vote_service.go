package apiservice

import (
	"context"
	"math/big"

	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/api"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-core/ioctl/util"
)

type AccountVoteService struct {
	api.UnimplementedAccountVoteServiceServer
}

type voteRow struct {
	StakeAmount, VoteWeight string
}

//curl -d '{"address": ["io1kp8alrznh2alvld3e34awq8aku7llqjjv9vkxk", "io1f8jdhlux5tja5f79gry6n6xjcdexufu0rl2hxj"], "height":5161495 }' http://127.0.0.1:7778/api.AccountVoteService.GetVoteByHeight
func (s *AccountVoteService) GetVoteByHeight(ctx context.Context, req *api.AccountVoteRequest) (*api.AccountVoteResponse, error) {
	resp := &api.AccountVoteResponse{
		Height: req.GetHeight(),
	}
	db := db.DB()
	height := req.GetHeight()
	for _, addr := range req.GetAddress() {
		if addr[:2] == "0x" || addr[:2] == "0X" {
			add, err := address.FromHex(addr)
			if err != nil {
				return nil, err
			}

			addr = add.String()
		}
		var row voteRow
		query := "SELECT sum(create_stake_amount)-sum(un_stake_amount) as stake_amount,sum(create_stake_vote_weight-un_stake_vote_weight) as vote_weight from account_vote WHERE block_height<=? and address=?"
		err := db.Raw(query, height, addr).First(&row).Error
		if err != nil {
			return nil, err
		}
		stakeAmount, ok := big.NewInt(0).SetString(row.StakeAmount, 10)
		if !ok {
			stakeAmount = big.NewInt(0)
		}
		resp.StakeAmount = append(resp.StakeAmount, util.RauToString(stakeAmount, util.IotxDecimalNum))
		voteWeight, ok := big.NewInt(0).SetString(row.VoteWeight, 10)
		if !ok {
			voteWeight = big.NewInt(0)
		}
		resp.VoteWeight = append(resp.VoteWeight, util.RauToString(voteWeight, util.IotxDecimalNum))
	}

	return resp, nil
}
