package verifymigration

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/urfave/cli/v2"
	"gorm.io/gorm"
)

var VerifyCandidateCmd = &cli.Command{
	Name:        "candidate",
	Usage:       "candidate",
	Description: "candidate will verify the clickhouse from postgres by blkNum",
	Action:      verifyCandidate,
}

func verifyCandidate(c *cli.Context) error {
	chConn, pg, err := connectDatabase(c)
	if err != nil {
		return err
	}
	start := c.Uint64("start")
	end := c.Uint64("end")
	batchSize := 2000
	for i := start; i <= end; {
		size := uint64(batchSize)
		if i+size-1 > end {
			size = end - i + 1
		}
		if err := verifyCandidateInRange(c.Context, pg, chConn, i, size); err != nil {
			return errors.Wrap(err, "failed to verify action_execution")
		}
		i = i + size
	}
	return nil
}

type candidate struct {
	ID              uint64          `gorm:"primary_key;" sql:"type:bigint"`
	BlockHeight     uint64          `gorm:"not null;unsigned;index" sql:"type:bigint"`
	Name            string          `gorm:"size:42;not null;default:'';index"`
	OperatorAddress string          `gorm:"size:42;not null;default:'';"`
	RewardAddress   string          `gorm:"size:42;not null;default:'';"`
	OwnerAddress    string          `gorm:"size:42;not null;default:'';"`
	CandidateID     string          `gorm:"size:42;not null;default:'';"`
	Amount          decimal.Decimal `gorm:"type:decimal(42,0);not null;default:0;"`
	Duration        uint32          `gorm:"not null;" sql:"type:bigint"`
	ActType         string
	AutoStake       bool
	Payload         []byte
}

func (candidate) TableName() string {
	return "candidate"
}

func verifyCandidateInRange(ctx context.Context, pg *gorm.DB, ch driver.Conn, start, size uint64) error {
	fmt.Println("\nverify candidate from", start, "to", start+size-1, ":")
	ches := make([]models.Candidate, 0)
	err := ch.Select(ctx, &ches,
		"SELECT * FROM candidate WHERE block_height >= ? AND block_height < ?",
		start, start+size)
	if err != nil {
		return errors.Wrap(err, "failed to select candidate from clickhouse")
	}

	pges := make([]*candidate, 0)
	err = pg.Model(&candidate{}).Where("block_height >= ? AND block_height < ?", start, start+size).Order("block_height").Find(&pges).Error
	if err != nil {
		return errors.Wrap(err, "failed to select candidate from postgres")
	}

	fmt.Printf("\tclickhouse count: %d, postgres count: %d\n", len(ches), len(pges))
	isConsistent := len(ches) == len(pges)
	chesMap := make(map[uint64]map[string]*models.Candidate)
	for _, ae := range ches {
		if _, ok := chesMap[ae.BlockHeight]; !ok {
			chesMap[ae.BlockHeight] = make(map[string]*models.Candidate)
		}
		chesMap[ae.BlockHeight][ae.CandidateID] = &ae
	}
	for _, pge := range pges {
		cheM, ok := chesMap[pge.BlockHeight]
		if !ok {
			fmt.Printf("\tcandidate not found in clickhouse: %s at %d\n", pge.CandidateID, pge.BlockHeight)
			isConsistent = false
			continue
		}
		che, ok := cheM[pge.CandidateID]
		if !ok {
			fmt.Printf("\tcandidate not found in clickhouse: %s at %d\n", pge.CandidateID, pge.BlockHeight)
			isConsistent = false
			continue
		}
		if pge.BlockHeight != che.BlockHeight ||
			pge.Name != che.Name ||
			!addressEqual(pge.OperatorAddress, che.OperatorAddress) ||
			!addressEqual(pge.RewardAddress, che.RewardAddress) ||
			!addressEqual(pge.OwnerAddress, che.OwnerAddress) ||
			!addressEqual(pge.CandidateID, che.CandidateID) ||
			pge.Amount.String() != che.Amount ||
			pge.Duration != che.Duration ||
			pge.ActType != che.ActType ||
			pge.AutoStake != che.AutoStake ||
			string(pge.Payload) != string(che.Payload) {
			fmt.Printf("\tcandidate not match: %v, %v\n", pge, che)
			isConsistent = false
		}
	}
	if isConsistent {
		fmt.Println("\tcandidate is consistent")
	}
	return nil
}
