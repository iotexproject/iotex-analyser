package main

import (
	"context"
	"encoding/hex"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/v2/blockchain/block"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const VERSION = "2.0.0"

type tokenPlugin struct {
}

func (b tokenPlugin) Name() string {
	return "erc1155_721_holder"
}

func (b tokenPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b tokenPlugin) Start(ctx context.Context) error {
	if err := initAddress(); err != nil {
		return errors.Wrap(err, "cannot init address")
	}
	if err := db.AutoMigrate(b.Name(),
		&models.Erc1155721Holder{}); err != nil {
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
			actionHash := hex.EncodeToString(receipt.ActionHash[:])
			for _, log := range receipt.Logs() {
				if log.Address == "" || len(log.Topics) < 2 {
					continue
				}
				data := hex.EncodeToString(log.Data)
				var topics string
				for _, t := range log.Topics {
					topics += hex.EncodeToString(t[:])
				}
				if isErc721(log.Address, topics, data) {
					switch log.Topics[0] {
					case Transfer:
						event := struct {
							From    common.Address
							To      common.Address
							TokenId *big.Int
						}{}
						err := kernel.UnpackLog(erc721ABI, &event, "Transfer", log)
						if err != nil {
							return errors.WithMessagef(err, "failed to unpack Transfer on action: %s", actionHash)
						}
						tokenID := decimal.NewFromBigInt(event.TokenId, 0)
						fromAddr, _ := address.FromBytes(event.From.Bytes())
						toAddr, _ := address.FromBytes(event.To.Bytes())
						//mint
						if fromAddr.String() == address.ZeroAddress {
							model := models.Erc1155721Holder{
								ContractAddress: log.Address,
								Holder:          toAddr.String(),
								ErcType:         721,
								TokenID:         tokenID,
							}
							if err := tx.Clauses(clause.OnConflict{
								Columns:   []clause.Column{{Name: "contract_address"}, {Name: "holder"}, {Name: "token_id"}},
								DoUpdates: clause.Assignments(map[string]interface{}{"token_value": 0}),
							}).Create(&model).Error; err != nil {
								return err
							}
							//burn
						} else if toAddr.String() == address.ZeroAddress {
							if err := tx.Where("contract_address = ? and token_id= ?", log.Address, tokenID).Delete(&models.Erc1155721Holder{}).Error; err != nil {
								return err
							}
							//transfer
						} else {
							if err := tx.Where("contract_address = ? and holder=? and token_id= ?", log.Address, fromAddr.String(), tokenID).Delete(&models.Erc1155721Holder{}).Error; err != nil {
								return err
							}
							model := models.Erc1155721Holder{
								ContractAddress: log.Address,
								Holder:          toAddr.String(),
								ErcType:         721,
								TokenID:         tokenID,
							}
							if err := tx.Clauses(clause.OnConflict{
								Columns:   []clause.Column{{Name: "contract_address"}, {Name: "holder"}, {Name: "token_id"}},
								DoUpdates: clause.Assignments(map[string]interface{}{"token_value": 0}),
							}).Create(&model).Error; err != nil {
								return err
							}

						}
					}
				}
				if isErc1155(log.Address, topics, data) {
					var tokenIDs, tokenVals []*big.Int
					var fromAddr, toAddr address.Address
					switch log.Topics[0] {
					//TransferBatch(address indexed operator, address indexed from, address indexed to, uint256[] ids, uint256[] values)
					case HashTransferBatch:
						event := struct {
							Operator common.Address
							From     common.Address
							To       common.Address
							Ids      []*big.Int
							Values   []*big.Int
						}{}
						err := kernel.UnpackLog(erc1155ABI, &event, "TransferBatch", log)
						if err != nil {
							return err
						}
						fromAddr, _ = address.FromBytes(event.From.Bytes())
						toAddr, _ = address.FromBytes(event.To.Bytes())
						tokenIDs = event.Ids
						tokenVals = event.Values

					//TransferSingle(address indexed operator, address indexed from, address indexed to, uint256 id, uint256 value)
					case HashTransferSingle:
						event := struct {
							Operator common.Address
							From     common.Address
							To       common.Address
							Id       *big.Int
							Value    *big.Int
						}{}
						err := kernel.UnpackLog(erc1155ABI, &event, "TransferSingle", log)
						if err != nil {
							return err
						}
						fromAddr, _ = address.FromBytes(event.From.Bytes())
						toAddr, _ = address.FromBytes(event.To.Bytes())

						tokenIDs = []*big.Int{event.Id}
						tokenVals = []*big.Int{event.Value}
					default:
						continue

					}
					for idx, tokenID := range tokenIDs {
						tokenIDDec := decimal.NewFromBigInt(tokenID, 0)
						tokenValDec := decimal.NewFromBigInt(tokenVals[idx], 0)
						//mint
						if fromAddr.String() == address.ZeroAddress {
							model := models.Erc1155721Holder{
								ContractAddress: log.Address,
								Holder:          toAddr.String(),
								ErcType:         1155,
								TokenID:         tokenIDDec,
								TokenValue:      tokenValDec,
							}
							if err := tx.Clauses(clause.OnConflict{
								Columns:   []clause.Column{{Name: "contract_address"}, {Name: "holder"}, {Name: "token_id"}},
								DoUpdates: clause.Assignments(map[string]interface{}{"token_value": gorm.Expr("erc1155_721_holders.token_value + ?", tokenValDec)}),
							}).Create(&model).Error; err != nil {
								return err
							}
							//burn
						} else if toAddr.String() == address.ZeroAddress {
							var tokenOldVal string
							if err := tx.Model(&models.Erc1155721Holder{}).Select("token_value").Where("contract_address = ? and holder=? and token_id= ?", log.Address, fromAddr.String(), tokenIDDec).Scan(&tokenOldVal).Error; err != nil {
								return err
							}
							tokenOldValDec, _ := decimal.NewFromString(tokenOldVal)
							if tokenOldValDec.Equal(tokenValDec) {
								if err := tx.Where("contract_address = ? and holder=? and token_id= ?", log.Address, fromAddr.String(), tokenIDDec).Delete(&models.Erc1155721Holder{}).Error; err != nil {
									return err
								}
							} else {
								if err := tx.Model(&models.Erc1155721Holder{}).Where("contract_address = ? and holder=? and token_id= ?", log.Address, fromAddr.String(), tokenIDDec).Update("token_value", gorm.Expr("erc1155_721_holders.token_value - ?", tokenValDec)).Error; err != nil {
									return err
								}
							}

							//transfer
						} else {
							var tokenOldVal string
							if err := tx.Model(&models.Erc1155721Holder{}).Select("token_value").Where("contract_address = ? and holder=? and token_id= ?", log.Address, fromAddr.String(), tokenIDDec).Scan(&tokenOldVal).Error; err != nil {
								return err
							}
							tokenOldValDec, _ := decimal.NewFromString(tokenOldVal)
							if tokenOldValDec.Equal(tokenValDec) {
								if err := tx.Where("contract_address = ? and holder=? and token_id= ?", log.Address, fromAddr.String(), tokenIDDec).Delete(&models.Erc1155721Holder{}).Error; err != nil {
									return err
								}
							} else {
								if err := tx.Model(&models.Erc1155721Holder{}).Where("contract_address = ? and holder=? and token_id= ?", log.Address, fromAddr.String(), tokenIDDec).Update("token_value", gorm.Expr("erc1155_721_holders.token_value - ?", tokenValDec)).Error; err != nil {
									return err
								}
							}
							model := models.Erc1155721Holder{
								ContractAddress: log.Address,
								Holder:          toAddr.String(),
								ErcType:         1155,
								TokenID:         tokenIDDec,
								TokenValue:      tokenValDec,
							}
							if err := tx.Clauses(clause.OnConflict{
								Columns:   []clause.Column{{Name: "contract_address"}, {Name: "holder"}, {Name: "token_id"}},
								DoUpdates: clause.Assignments(map[string]interface{}{"token_value": gorm.Expr("erc1155_721_holders.token_value + ?", tokenValDec)}),
							}).Create(&model).Error; err != nil {
								return err
							}

						}
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
