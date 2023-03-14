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
	"github.com/iotexproject/iotex-core/blockchain/block"
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
							tokenURI, err := readERC721URI(log.Address, event.TokenId)
							if err != nil {
								return errors.WithMessagef(err, "failed to read ERC721 URI on action: %s", actionHash)
							}
							model := models.Erc1155721Holder{
								ContractAddress: log.Address,
								Holder:          toAddr.String(),
								ErcType:         721,
								TokenID:         tokenID,
								TokenURI:        tokenURI,
							}
							if err := tx.Clauses(clause.OnConflict{
								Columns:   []clause.Column{{Name: "contract_address"}, {Name: "token_id"}},
								DoUpdates: clause.Assignments(map[string]interface{}{"holder": toAddr.String(), "token_uri": tokenURI}),
							}).Create(&model).Error; err != nil {
								return err
							}
							//born
						} else if toAddr.String() == address.ZeroAddress {
							if err := tx.Where("contract_address = ? and token_id= ?", log.Address, tokenID).Delete(&models.Erc1155721Holder{}).Error; err != nil {
								return err
							}
							//transfer
						} else {
							model := models.Erc1155721Holder{
								ContractAddress: log.Address,
								Holder:          toAddr.String(),
								ErcType:         721,
								TokenID:         tokenID,
							}
							if err := tx.Clauses(clause.OnConflict{
								Columns:   []clause.Column{{Name: "contract_address"}, {Name: "token_id"}},
								DoUpdates: clause.Assignments(map[string]interface{}{"holder": toAddr.String()}),
							}).Create(&model).Error; err != nil {
								return err
							}

						}
					}
				}
				if isErc1155(log.Address, topics, data) {
					var tokenIDs []*big.Int
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
					default:
						continue

					}
					for _, tokenID := range tokenIDs {
						tokenIDDec := decimal.NewFromBigInt(tokenID, 0)
						//mint
						if fromAddr.String() == address.ZeroAddress {
							tokenURI, err := readERC1155URI(log.Address, tokenID)
							if err != nil {
								return errors.WithMessagef(err, "failed to read ERC1155 URI on action: %s", actionHash)
							}
							model := models.Erc1155721Holder{
								ContractAddress: log.Address,
								Holder:          toAddr.String(),
								ErcType:         1155,
								TokenID:         tokenIDDec,
								TokenURI:        tokenURI,
							}
							if err := tx.Clauses(clause.OnConflict{
								Columns:   []clause.Column{{Name: "contract_address"}, {Name: "token_id"}},
								DoUpdates: clause.Assignments(map[string]interface{}{"holder": toAddr.String(), "token_uri": tokenURI}),
							}).Create(&model).Error; err != nil {
								return err
							}
							//born
						} else if toAddr.String() == address.ZeroAddress {
							if err := tx.Where("contract_address = ? and token_id= ?", log.Address, tokenID).Delete(&models.Erc1155721Holder{}).Error; err != nil {
								return err
							}
							//transfer
						} else {
							model := models.Erc1155721Holder{
								ContractAddress: log.Address,
								Holder:          toAddr.String(),
								ErcType:         1155,
								TokenID:         tokenIDDec,
							}
							if err := tx.Clauses(clause.OnConflict{
								Columns:   []clause.Column{{Name: "contract_address"}, {Name: "token_id"}},
								DoUpdates: clause.Assignments(map[string]interface{}{"holder": toAddr.String()}),
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
