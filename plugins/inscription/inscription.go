package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/iotexproject/go-pkgs/hash"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/iotexproject/iotex-analyser/plugin"
	"github.com/iotexproject/iotex-core/action"
	"github.com/iotexproject/iotex-core/blockchain/block"
	slog "github.com/iotexproject/iotex-core/pkg/log"
	"github.com/iotexproject/iotex-proto/golang/iotextypes"
	"github.com/pkg/errors"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
	"gorm.io/gorm"
)

var (
	batchSize = 200
)

type tokenPlugin struct {
}

func (b tokenPlugin) Name() string {
	return "inscription"
}

func (b tokenPlugin) Type() plugin.Type {
	return plugin.TypeStandard
}

func (b tokenPlugin) Start(ctx context.Context) error {
	if err := db.AutoMigrate(b.Name(),
		&models.InscriptionRaw{},
		&models.Inscription{},
		&models.InscriptionTransfer{},
		&models.InscriptionHolder{},
		&models.InscriptionToken{},
		&models.InscriptionTokenTransaction{},
		&models.InscriptionTokenHolder{},
	); err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	height, err := db.GetIndexHeight(b.Name())
	if err != nil {
		return errors.Wrapf(err, "failed to start plugin %s", b.Name())
	}
	startHeight := uint64(0)
	if cfgData, ok := kernel.GetPluginConfigCtx(ctx); ok {
		cfg := &Config{}
		if err = yaml.Unmarshal(cfgData, cfg); err != nil {
			return errors.Wrapf(err, "failed to unmarshal plugin config: plugin %s, config %s", b.Name(), string(cfgData))
		} else {
			startHeight = cfg.StartHeight
			slog.L().Info("read plugin config success", zap.String("plugin", b.Name()), zap.Any("config", cfg))
		}
	}
	if height < startHeight && startHeight > 0 {
		return db.UpdateIndexHeight(b.Name(), startHeight-1)
	}
	return nil
}

func (b tokenPlugin) PutBlock(ctx context.Context, blk *block.Block) error {
	return db.DB().Transaction(func(gormTx *gorm.DB) error {
		return b.putBlock(ctx, gormTx, blk)
	})
}

func (b tokenPlugin) putBlock(ctx context.Context, gormTx *gorm.DB, blk *block.Block) error {
	getActReceiptFun := func(blk *block.Block, actHash hash.Hash256) *action.Receipt {
		for _, receipt := range blk.Receipts {
			if receipt.ActionHash == actHash {
				return receipt
			}
		}
		return nil
	}
	inscriptRaws := make([]*models.InscriptionRaw, 0)
	inscripts := make([]*models.Inscription, 0)
	inscriptTransfers := make([]*models.InscriptionTransfer, 0)
	inscriptionHolders := make([]*models.InscriptionHolder, 0)

	inscriptMap := make(map[string]*models.Inscription, 0)
	inscriptionHolderMap := make(map[string]*models.InscriptionHolder, 0)

	inscriptionTokens := make([]*models.InscriptionToken, 0)
	inscriptionTokenTransactions := make([]*models.InscriptionTokenTransaction, 0)
	inscriptionTokenHolders := make([]*models.InscriptionTokenHolder, 0)

	inscriptionTokenMap := make(map[string]*models.InscriptionToken, 0)
	inscriptionTokenHolderMap := make(map[string]*models.InscriptionTokenHolder, 0)

	for index, act := range blk.Actions {
		actHash, err := act.Hash()
		if err != nil {
			continue
		}
		slog.L().Debug("handle action", zap.Any("hash", hex.EncodeToString(actHash[:])), zap.Any("block", blk.Height()))
		receipt := getActReceiptFun(blk, actHash)
		if receipt == nil {
			slog.L().Debug("skip action: no receipt", zap.Any("hash", hex.EncodeToString(actHash[:])))
			continue
		}
		if receipt.Status != uint64(iotextypes.ReceiptStatus_Success) {
			slog.L().Debug("skip action: action failed", zap.Any("hash", hex.EncodeToString(actHash[:])))
			continue
		}
		tx, ok := act.Action().(action.EthCompatibleAction)
		if !ok {
			slog.L().Debug("skip action: not eth-compatible", zap.Any("hash", hex.EncodeToString(actHash[:])))
			continue
		}
		ethTx, err := tx.ToEthTx()
		if err != nil {
			slog.L().Debug("skip action: to eth error", zap.Any("hash", hex.EncodeToString(actHash[:])), zap.Error(err))
			continue
		}
		data := ethTx.Data()
		if len(data) == 0 {
			slog.L().Debug("skip action: empty data", zap.Any("hash", hex.EncodeToString(actHash[:])))
			continue
		}
		text, err := bytesToUTF8(data)
		if err != nil {
			slog.L().Debug("skip action: not a utf8 string", zap.Any("hash", hex.EncodeToString(actHash[:])))
			continue
		}
		fromAddr, _ := address.FromBytes(act.SenderAddress().Bytes())
		toAddr, _ := address.FromBytes(ethTx.To().Bytes())
		actHashStr := hex.EncodeToString(actHash[:])
		inscriptRaws = append(inscriptRaws, &models.InscriptionRaw{
			BlockHeight:      blk.Height(),
			ActionHash:       actHashStr,
			TransactionIndex: uint64(index),
			Sender:           fromAddr.String(),
			Recipient:        toAddr.String(),
			Timestamp:        time.Unix(blk.Timestamp().Unix(), 0),
			RawData:          text,
		})

		// validate inscription
		uri, err := ParseDataURI(text)
		// inscription transfer
		if err != nil {
			// EOA transfer
			inscript, err := getInscriptionByHash(text, inscriptMap)
			if err != nil {
				slog.L().Debug("skip action: get inscription error", zap.Any("hash", hex.EncodeToString(actHash[:])), zap.Error(err))
				continue
			}
			inscriptTransfers = append(inscriptTransfers, &models.InscriptionTransfer{
				BlockHeight:      blk.Height(),
				ActionHash:       actHashStr,
				TransactionIndex: uint64(index),
				Sender:           fromAddr.String(),
				Recipient:        toAddr.String(),
				Timestamp:        time.Unix(blk.Timestamp().Unix(), 0),
				InscriptionHash:  inscript.ActionHash,
			})

			// update inscription holder
			inscriptionHolder, err := getInscriptionHolderByHash(text, inscriptionHolderMap)
			if err != nil {
				slog.L().Debug("skip action: get inscription holder error", zap.Any("hash", hex.EncodeToString(actHash[:])), zap.Error(err))
				continue
			}
			if inscriptionHolder.Owner == fromAddr.String() && inscriptionHolder.IsTransfer == false {
				inscriptionHolder.IsTransfer = true
				inscriptionHolder.Timestamp = time.Unix(blk.Timestamp().Unix(), 0)
				// transfer Owner
				toOwner := &models.InscriptionHolder{
					Owner:           toAddr.String(),
					InscriptionHash: text,
					IsTransfer:      false,
					Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
				}
				// set default ID for insert
				inscriptionHolder.ID = 0
				inscriptionHolders = append(inscriptionHolders, inscriptionHolder, toOwner)
				// update inscriptionHolderMap
				inscriptionHolderMap[text] = toOwner
			} else {
				slog.L().Debug("skip action: inscription owner error", zap.Any("hash", hex.EncodeToString(actHash[:])))
			}
			continue
		}
		// inscription create
		inscription := &models.Inscription{
			BlockHeight:      blk.Height(),
			ActionHash:       actHashStr,
			TransactionIndex: uint64(index),
			MIMEType:         uri.MIMEType,
			Parameters:       uri.Parameters,
			Extension:        uri.Extension,
			Data:             uri.Data,
		}
		inscripts = append(inscripts, inscription)
		inscriptMap[actHashStr] = inscription

		// update inscription holder
		inscriptionHolder := &models.InscriptionHolder{
			Owner:           fromAddr.String(),
			InscriptionHash: actHashStr,
			IsTransfer:      false,
			Timestamp:       time.Unix(blk.Timestamp().Unix(), 0),
		}
		inscriptionHolders = append(inscriptionHolders, inscriptionHolder)
		inscriptionHolderMap[actHashStr] = inscriptionHolder

		// parse uri.data
		p := gjson.Get(uri.Data, "p").String()
		tick := gjson.Get(uri.Data, "tick").String()
		op := gjson.Get(uri.Data, "op").String()
		switch op {
		case "deploy":
			inscriptionToken, err := getInscriptionTokenByPAndTick(p, tick, inscriptionTokenMap)
			if err == nil {
				inscriptionToken = &models.InscriptionToken{
					BlockHeight: blk.Height(),
					ActionHash:  actHashStr,
					Owner:       fromAddr.String(),
					P:           p,
					Tick:        tick,
					Op:          op,
					Max:         gjson.Get(uri.Data, "max").Uint(),
					Lim:         gjson.Get(uri.Data, "lim").Uint(),
					Description: gjson.Get(uri.Data, "description").String(),
					Verified:    false,
					Timestamp:   time.Unix(blk.Timestamp().Unix(), 0),
				}
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				inscriptionToken = &models.InscriptionToken{
					BlockHeight: blk.Height(),
					ActionHash:  actHashStr,
					Owner:       fromAddr.String(),
					P:           p,
					Tick:        tick,
					Op:          op,
					Max:         gjson.Get(uri.Data, "max").Uint(),
					Lim:         gjson.Get(uri.Data, "lim").Uint(),
					Description: gjson.Get(uri.Data, "description").String(),
					Verified:    true,
					Timestamp:   time.Unix(blk.Timestamp().Unix(), 0),
				}
				inscriptionTokenMap[getTokenKey(p, tick)] = inscriptionToken
			} else {
				slog.L().Debug("skip action: get inscription token error", zap.Any("hash", hex.EncodeToString(actHash[:])), zap.Error(err))
				continue
			}
			// set default ID for insert
			inscriptionToken.ID = 0
			inscriptionTokens = append(inscriptionTokens, inscriptionToken)
		case "mint":
			tokenHolder, err := getTokenHolderInfoByHolder(fromAddr.String(), inscriptionTokenHolderMap)
			if err == nil {
				tokenHolder.Amt = tokenHolder.Amt + gjson.Get(uri.Data, "amt").Uint()
				tokenHolder.Timestamp = time.Unix(blk.Timestamp().Unix(), 0)
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				tokenHolder = &models.InscriptionTokenHolder{
					Owner:     fromAddr.String(),
					P:         p,
					Tick:      tick,
					Amt:       gjson.Get(uri.Data, "amt").Uint(),
					Timestamp: time.Unix(blk.Timestamp().Unix(), 0),
				}
				inscriptionTokenHolderMap[fromAddr.String()] = tokenHolder
			} else {
				slog.L().Debug("skip action: get inscription token holder error", zap.Any("hash", hex.EncodeToString(actHash[:])), zap.Error(err))
				continue
			}
			// set default ID for insert
			tokenHolder.ID = 0
			inscriptionTokenHolders = append(inscriptionTokenHolders, tokenHolder)
		case "transfer":
			fromHolder, err := getTokenHolderInfoByHolder(fromAddr.String(), inscriptionTokenHolderMap)
			if err != nil {
				slog.L().Debug("skip action: get inscription fromHolder error", zap.Any("hash", hex.EncodeToString(actHash[:])), zap.Error(err))
				// TODO check not found record
				continue
			}
			amt := gjson.Get(uri.Data, "amt").Uint()
			if fromHolder.Amt-amt < 0 {
				slog.L().Debug("skip action: the balance not enough", zap.Any("hash", hex.EncodeToString(actHash[:])))
				// TODO insert transactions
				continue
			}
			fromHolder.Amt = fromHolder.Amt - amt

			toHolder, err := getTokenHolderInfoByHolder(toAddr.String(), inscriptionTokenHolderMap)
			if err == nil {
				toHolder.Amt = toHolder.Amt + amt
				toHolder.Timestamp = time.Unix(blk.Timestamp().Unix(), 0)
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				toHolder = &models.InscriptionTokenHolder{
					Owner:     toAddr.String(),
					P:         p,
					Tick:      tick,
					Amt:       amt,
					Timestamp: time.Unix(blk.Timestamp().Unix(), 0),
				}
				inscriptionTokenHolderMap[toAddr.String()] = toHolder
			} else {
				slog.L().Debug("skip action: get inscription toHolder error", zap.Any("hash", hex.EncodeToString(actHash[:])), zap.Error(err))
				continue
			}
			// set default ID for insert
			fromHolder.ID = 0
			toHolder.ID = 0
			inscriptionTokenHolders = append(inscriptionTokenHolders, fromHolder, toHolder)
		default:
			slog.L().Debug("skip action: not support method", zap.Any("hash", hex.EncodeToString(actHash[:])))
		}
		// update transfer
		inscriptionTokenTransactions = append(inscriptionTokenTransactions, &models.InscriptionTokenTransaction{
			BlockHeight:      blk.Height(),
			ActionHash:       actHashStr,
			TransactionIndex: uint64(index),
			Sender:           fromAddr.String(),
			Recipient:        toAddr.String(),
			Timestamp:        time.Unix(blk.Timestamp().Unix(), 0),
			Method:           op,
		})
	}
	if err := gormTx.CreateInBatches(inscriptRaws, batchSize).Error; err != nil {
		return errors.Wrapf(err, "failed to put block %d", blk.Height())
	}
	if err := gormTx.CreateInBatches(inscripts, batchSize).Error; err != nil {
		return errors.Wrapf(err, "failed to put block %d", blk.Height())
	}
	if err := gormTx.CreateInBatches(inscriptTransfers, batchSize).Error; err != nil {
		return errors.Wrapf(err, "failed to put block %d", blk.Height())
	}
	if err := gormTx.CreateInBatches(inscriptionHolders, batchSize).Error; err != nil {
		return errors.Wrapf(err, "failed to put block %d", blk.Height())
	}

	if err := gormTx.CreateInBatches(inscriptionTokens, batchSize).Error; err != nil {
		return errors.Wrapf(err, "failed to put block %d", blk.Height())
	}
	if err := gormTx.CreateInBatches(inscriptionTokenHolders, batchSize).Error; err != nil {
		return errors.Wrapf(err, "failed to put block %d", blk.Height())
	}
	if err := gormTx.CreateInBatches(inscriptionTokenTransactions, batchSize).Error; err != nil {
		return errors.Wrapf(err, "failed to put block %d", blk.Height())
	}

	// TODO: move to plugin framework to update index height
	return db.UpdateIndexHeightByTx(gormTx, b.Name(), blk.Height())
}

func (b tokenPlugin) Stop(ctx context.Context) error {
	return nil
}

func (b tokenPlugin) Version() string {
	return "0.0.1"
}

// exported
var Plugin = tokenPlugin{}

func bytesToUTF8(data []byte) (string, error) {
	if !utf8.Valid(data) {
		return "", errors.New("not a utf8 string")
	}
	res := string(data)
	// remove null bytes
	res = strings.Replace(res, "\x00", "", -1)
	return res, nil
}

func getInscriptionByHash(inscriptionHash string, inscriptionMap map[string]*models.Inscription) (*models.Inscription, error) {
	if !isHash(inscriptionHash) {
		return nil, errors.New("not a hex string")
	}

	if inscription, ok := inscriptionMap[inscriptionHash]; ok {
		return inscription, nil
	}

	inscription := &models.Inscription{}
	if err := db.DB().First(inscription, "action_hash = ?", inscriptionHash).Error; err != nil {
		return nil, err
	}
	inscriptionMap[inscriptionHash] = inscription
	return inscription, nil
}

func isHash(s string) bool {
	if len(s) != 64 {
		return false
	}
	matched, _ := regexp.MatchString(`^[a-fA-F0-9]*$`, s)
	return matched
}

func getInscriptionHolderByHash(inscriptionHash string, inscriptionHolderMap map[string]*models.InscriptionHolder) (*models.InscriptionHolder, error) {
	if inscriptionHolder, ok := inscriptionHolderMap[inscriptionHash]; ok {
		return inscriptionHolder, nil
	}

	inscriptionHolder := &models.InscriptionHolder{}
	if err := db.DB().Order("id desc").First(inscriptionHolder, "inscription_hash = ?", inscriptionHash).Error; err != nil {
		return nil, err
	}
	inscriptionHolderMap[inscriptionHash] = inscriptionHolder
	return inscriptionHolder, nil
}

func getInscriptionTokenByPAndTick(p, tick string, inscriptionTokenMap map[string]*models.InscriptionToken) (*models.InscriptionToken, error) {
	if token, ok := inscriptionTokenMap[getTokenKey(p, tick)]; ok {
		return token, nil
	}

	token := &models.InscriptionToken{}
	if err := db.DB().First(token, "p = ? and tick = ?", p, tick).Error; err != nil {
		return nil, err
	}
	inscriptionTokenMap[getTokenKey(p, tick)] = token
	return token, nil
}

func getTokenKey(p, tick string) string {
	return fmt.Sprintf("%s-%s", p, tick)
}

func getTokenHolderInfoByHolder(holder string, inscriptionTokenHolderMap map[string]*models.InscriptionTokenHolder) (*models.InscriptionTokenHolder, error) {
	if tokenHolder, ok := inscriptionTokenHolderMap[holder]; ok {
		return tokenHolder, nil
	}

	tokenHolder := &models.InscriptionTokenHolder{}
	if err := db.DB().Order("id desc").First(tokenHolder, "owner = ?", holder).Error; err != nil {
		return nil, err
	}
	inscriptionTokenHolderMap[holder] = tokenHolder
	return tokenHolder, nil
}
