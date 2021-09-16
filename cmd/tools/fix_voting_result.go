package tools

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"log"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/urfave/cli/v2"
	"gorm.io/gorm"
)

type VotingResult struct {
	EpochNumber               uint64
	DelegateName              string
	OperatorAddress           string
	RewardAddress             string
	TotalWeightedVotes        string
	SelfStaking               string
	BlockRewardPercentage     float64
	EpochRewardPercentage     float64
	FoundationBonusPercentage float64
	StakingAddress            string
}

var FixVotingResult = &cli.Command{
	Name:        "fixVotingResult",
	Usage:       "fixVotingResult --epochNum <epochNum>",
	Description: `fix voting result data`,
	Flags: []cli.Flag{
		&cli.Uint64Flag{
			Name:  "epochNum",
			Usage: "epochNum epochNum",
		},
	},
	Action: fixVotingResult,
}

func getRowID(t *gorm.DB, query string, args ...interface{}) int64 {
	var id sql.NullInt64
	t.Raw(query, args...).Scan(&id)
	return id.Int64
}

func getName(t *gorm.DB, query string, args ...interface{}) string {
	var name string
	t.Raw(query, args...).Scan(&name)
	return name
}

func fixVotingResult(c *cli.Context) error {
	epochNum := c.Uint64("epochNum")
	if epochNum == 0 {
		return errors.New("missing --epochNum <epochNum>")
	}

	db, err := db.Connect()
	if err != nil {
		return err
	}
	rows, err := db.Table("voting_result").Where("epoch_number<=?", epochNum).Rows()
	if err != nil {
		return err
	}
	defer rows.Close()

	id := 0
	for rows.Next() {
		id = id + 1
		if id <= 1276870 {
			continue
		}
		var vr VotingResult
		db.ScanRows(rows, &vr)
		dname, err := hex.DecodeString(strings.Replace(strings.TrimSpace(vr.DelegateName), "\u0000", "", -1))

		if vr.DelegateName == "000000000000000000000000" {
			dname = []byte("-")
		}
		if err != nil {
			log.Println(strings.Replace(vr.OperatorAddress, "\u0000", "", -1))
			dn1 := getName(db, "select name from candidate where operator_address=?", strings.Replace(vr.OperatorAddress, "\u0000", "", -1))
			if dn1 == "" {
				dname = []byte(" ")
			} else {
				dname = []byte(dn1)
			}

		}
		ethStakingAddress := common.HexToAddress(strings.Replace(vr.StakingAddress, "\u0000", "", -1))
		ioStakingAddress, err := address.FromBytes(ethStakingAddress.Bytes())
		if err != nil {
			return errors.WithMessage(err, "ioStakingAddress")
		}
		TotalWeightedVotes, err := decimal.NewFromString(strings.Replace(vr.TotalWeightedVotes, "\u0000", "", -1))
		if err != nil {
			return errors.WithMessage(err, "TotalWeightedVotes")
		}
		SelfStaking, err := decimal.NewFromString(strings.Replace(vr.SelfStaking, "\u0000", "", -1))
		if err != nil {
			return errors.WithMessage(err, "SelfStaking")
		}
		BlockRewardPercentage := decimal.NewFromFloat(vr.BlockRewardPercentage)
		EpochRewardPercentage := decimal.NewFromFloat(vr.EpochRewardPercentage)
		FoundationBonusPercentage := decimal.NewFromFloat(vr.FoundationBonusPercentage)

		if string(dname) == "" {
			dname = []byte(" ")
		}
		//log.Printf("'%v' '%v'", dname, bytes.Trim(dname, "\u0000"))
		m := models.HermesVotingResult{
			ID:                        uint64(id),
			EpochNumber:               vr.EpochNumber,
			DelegateName:              string(bytes.Trim(dname, "\u0000")),
			OperatorAddress:           strings.Replace(vr.OperatorAddress, "\u0000", "", -1),
			RewardAddress:             strings.Replace(vr.RewardAddress, "\u0000", "", -1),
			StakingAddress:            ioStakingAddress.String(),
			TotalWeightedVotes:        TotalWeightedVotes,
			SelfStaking:               SelfStaking,
			BlockRewardPercentage:     BlockRewardPercentage,
			EpochRewardPercentage:     EpochRewardPercentage,
			FoundationBonusPercentage: FoundationBonusPercentage,
		}
		err = db.Create(&m).Error
		if err != nil {
			return errors.WithMessage(err, "Create")
		}
	}

	return err
}
