package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/holiman/uint256"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/urfave/cli/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var BackfillAuthorization = &cli.Command{
	Name:  "backfill-authorization",
	Usage: "Backfill authorization table from existing action_type.auth_list (type-4 txs)",
	Flags: []cli.Flag{
		&cli.IntFlag{Name: "batch", Value: 100, Usage: "rows to process per batch"},
		&cli.BoolFlag{Name: "recompute-validity", Usage: "Recompute the Valid column for existing authorization rows instead of inserting from action_type"},
		&cli.BoolFlag{Name: "force", Usage: "With --recompute-validity, also recompute rows that already have a Valid value (default: only NULL rows)"},
	},
	Action: backfillAuthorization,
}

func backfillAuthorization(c *cli.Context) error {
	gdb, err := db.Connect()
	if err != nil {
		return fmt.Errorf("connect db: %w", err)
	}
	if err := gdb.AutoMigrate(&models.Authorization{}); err != nil {
		return fmt.Errorf("automigrate: %w", err)
	}

	if c.Bool("recompute-validity") {
		return recomputeAuthorizationValidity(c, gdb)
	}

	batchSize := c.Int("batch")
	offset := 0
	total := 0

	type actionTypeRow struct {
		Hash        string
		BlockHeight uint64
		AuthList    []byte
	}

	for {
		var rows []actionTypeRow
		if err := gdb.
			Model(&models.ActionType{}).
			Select("hash, block_height, auth_list").
			Where("type = 4 AND auth_list IS NOT NULL").
			Order("block_height ASC").
			Limit(batchSize).
			Offset(offset).
			Scan(&rows).Error; err != nil {
			return fmt.Errorf("query action_type: %w", err)
		}
		if len(rows) == 0 {
			break
		}

		var auths []*models.Authorization
		for _, row := range rows {
			var authList []ethtypes.SetCodeAuthorization
			if err := json.Unmarshal(row.AuthList, &authList); err != nil {
				fmt.Printf("skip %s: failed to unmarshal auth_list: %v\n", row.Hash, err)
				continue
			}
			for i, auth := range authList {
				auths = append(auths, buildAuthorizationRow(row.Hash, row.BlockHeight, i, auth))
			}
		}

		if len(auths) > 0 {
			if err := gdb.Transaction(func(tx *gorm.DB) error {
				return tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "action_hash"}, {Name: "index"}},
					DoNothing: true,
				}).CreateInBatches(auths, 200).Error
			}); err != nil {
				return fmt.Errorf("insert authorizations: %w", err)
			}
			total += len(auths)
		}

		fmt.Printf("processed %d action_type rows (offset %d), %d authorization rows written so far\n",
			len(rows), offset, total)
		offset += len(rows)

		if len(rows) < batchSize {
			break
		}
	}

	fmt.Printf("done: %d authorization rows total\n", total)
	return nil
}

// recomputeAuthorizationValidity walks the authorization table and computes
// the Valid column via kernel.ComputeAuthorizationValidity (eth_getCode +
// eth_getTransactionCount on the configured archive endpoint). By default
// only rows with Valid IS NULL are touched; pass --force to recompute all.
func recomputeAuthorizationValidity(c *cli.Context, gdb *gorm.DB) error {
	ctx := c.Context
	batchSize := c.Int("batch")
	force := c.Bool("force")

	processed := 0
	updated := 0
	lastID := uint64(0)

	for {
		var rows []models.Authorization
		q := gdb.WithContext(ctx).
			Where("id > ?", lastID).
			Order("id ASC").
			Limit(batchSize)
		if !force {
			q = q.Where("valid IS NULL")
		}
		if err := q.Find(&rows).Error; err != nil {
			return fmt.Errorf("query authorization rows: %w", err)
		}
		if len(rows) == 0 {
			break
		}

		for _, row := range rows {
			lastID = row.ID
			processed++
			if row.Authority == "" {
				// Signature recovery had failed at index time → permanently invalid.
				if row.Valid == nil {
					invalid := false
					if err := gdb.WithContext(ctx).
						Model(&models.Authorization{}).
						Where("id = ?", row.ID).
						Update("valid", &invalid).Error; err != nil {
						return fmt.Errorf("update row %d: %w", row.ID, err)
					}
					updated++
				}
				continue
			}

			chainID, err := uint256.FromHex(row.ChainID)
			if err != nil {
				return fmt.Errorf("parse chain_id %q (row id=%d): %w", row.ChainID, row.ID, err)
			}
			nonceUint, err := uint256.FromHex(row.Nonce)
			if err != nil {
				return fmt.Errorf("parse nonce %q (row id=%d): %w", row.Nonce, row.ID, err)
			}
			authority := common.HexToAddress(row.Authority)

			valid, err := kernel.ComputeAuthorizationValidity(ctx, authority, row.BlockHeight, chainID, nonceUint.Uint64())
			if err != nil {
				return fmt.Errorf("compute validity (row id=%d): %w", row.ID, err)
			}

			if err := gdb.WithContext(ctx).
				Model(&models.Authorization{}).
				Where("id = ?", row.ID).
				Update("valid", &valid).Error; err != nil {
				return fmt.Errorf("update row %d: %w", row.ID, err)
			}
			updated++
		}

		fmt.Printf("processed %d rows (last id %d), %d updated so far\n", processed, lastID, updated)
		if len(rows) < batchSize {
			break
		}
	}

	fmt.Printf("done: processed %d rows, updated %d\n", processed, updated)
	return nil
}

func buildAuthorizationRow(actionHash string, blockHeight uint64, index int, auth ethtypes.SetCodeAuthorization) *models.Authorization {
	row := &models.Authorization{
		ActionHash:  actionHash,
		BlockHeight: blockHeight,
		Index:       index,
		ChainID:     auth.ChainID.Hex(),
		Address:     strings.ToLower(auth.Address.Hex()),
		Nonce:       fmt.Sprintf("0x%x", auth.Nonce),
		YParity:     fmt.Sprintf("0x%x", auth.V),
		R:           auth.R.Hex(),
		S:           auth.S.Hex(),
	}
	if authority, err := auth.Authority(); err == nil {
		row.Authority = strings.ToLower(authority.Hex())
	}
	return row
}
