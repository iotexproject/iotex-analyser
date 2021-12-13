package main

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/blockchain/block"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const VERSION = "2.0.0"

const (
	ShiBaContractAddress = "io186ngxd2tlrf4nnv7cmsgkkhv9ywhrkyq8rejue"
)

var successStatus = uint64(1)
var (
	shibaABI abi.ABI
)

type tokenPlugin struct {
}

/*
{
	"39509351": "increaseAllowance(address,uint256)",
	"6bc87c3a": "_liquidityFee()",
	"7d1db4a5": "_maxTxAmount()",
	"3b124fe7": "_taxFee()",
	"dd62ed3e": "allowance(address,address)",
	"095ea7b3": "approve(address,uint256)",
	"70a08231": "balanceOf(address)",
	"313ce567": "decimals()",
	"a457c2d7": "decreaseAllowance(address,uint256)",
	"3bd5d173": "deliver(uint256)",
	"437823ec": "excludeFromFee(address)",
	"52390c02": "excludeFromReward(address)",
	"b6c52324": "geUnlockTime()",
	"ea2f0b37": "includeInFee(address)",
	"3685d419": "includeInReward(address)",
	"5342acb4": "isExcludedFromFee(address)",
	"88f82020": "isExcludedFromReward(address)",
	"dd467064": "lock(uint256)",
	"06fdde03": "name()",
	"8da5cb5b": "owner()",
	"4549b039": "reflectionFromToken(uint256,bool)",
	"715018a6": "renounceOwnership()",
	"8ee88c53": "setLiquidityFeePercent(uint256)",
	"d543dbeb": "setMaxTxPercent(uint256)",
	"c49b9a80": "setSwapAndLiquifyEnabled(bool)",
	"061c82d0": "setTaxFeePercent(uint256)",
	"4a74bb02": "swapAndLiquifyEnabled()",
	"95d89b41": "symbol()",
	"2d838119": "tokenFromReflection(uint256)",
	"13114a9d": "totalFees()",
	"18160ddd": "totalSupply()",
	"a9059cbb": "transfer(address,uint256)",
	"23b872dd": "transferFrom(address,address,uint256)",
	"f2fde38b": "transferOwnership(address)",
	"49bd5a5e": "uniswapV2Pair()",
	"1694505e": "uniswapV2Router()",
	"a69df4b5": "unlock()"
}
*/
var (
	Transfer hash.Hash256
	Approval hash.Hash256
)

func initAddress() error {
	var err error
	//Transfer(address,address,uint256)
	Transfer, err = hash.HexStringToHash256("ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	if err != nil {
		return err
	}
	//Approval(address,address,uint256)
	Approval, err = hash.HexStringToHash256("8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925")
	if err != nil {
		return err
	}

	shibaABI, err = abi.JSON(strings.NewReader(ShiBaABI))
	if err != nil {
		return err
	}
	return nil
}

func (b tokenPlugin) Name() string {
	return "erc20"
}

func (b tokenPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b tokenPlugin) Start(ctx context.Context) error {
	if err := initAddress(); err != nil {
		return errors.Wrap(err, "cannot init address")
	}
	if err := db.AutoMigrate(b.Name(), &models.Erc20{}, &models.Erc20Holder{}); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	return nil
}

func (b tokenPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	err := db.DB().Transaction(func(tx *gorm.DB) error {

		for _, receipt := range blk.Receipts {
			if receipt.Status != successStatus {
				continue
			}
			for _, log := range receipt.Logs() {
				topics := log.Topics
				if log.Address == "" || len(topics) < 2 {
					continue
				}
				switch log.Address {
				case ShiBaContractAddress:
					switch topics[0] {
					/**
					 * Transfer(address indexed from, address indexed to, uint256 value);
					 */
					case Transfer:
						event := struct {
							From  common.Address
							To    common.Address
							Value *big.Int
						}{}
						err := UnpackLog(shibaABI, &event, "Transfer", log)
						if err != nil {
							return err
						}
						fmt.Printf("block: %d Transfer %v\n", blk.Height(), event)
						//Approval(address indexed owner, address indexed spender, uint256 value);
					case Approval:
						event := struct {
							Owner   common.Address
							Spender common.Address
							Value   *big.Int
						}{}
						err := UnpackLog(shibaABI, &event, "Approval", log)
						if err != nil {
							return err
						}
						fmt.Printf("block: %d Approval %v\n", blk.Height(), event)
					}
				}
			}
		}

		return db.UpdateIndexHeightByTx(tx, b.Name(), blk.Height())
	})

	return err
}

func (b tokenPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b tokenPlugin) Version() string {
	return VERSION
}

// exported
var Plugin = tokenPlugin{}
