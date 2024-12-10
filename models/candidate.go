package models

import (
	"context"
	"errors"
	"strings"

	"github.com/iotexproject/iotex-analyser/db"
)

var CandidateDDL = `CREATE TABLE IF NOT EXISTS candidate
(
    block_height UInt64 NOT NULL,
	name String NOT NULL,
    operator_address FixedString(41) NOT NULL,
	reward_address FixedString(41) NOT NULL,
	owner_address FixedString(41) NOT NULL,
	candidate_id FixedString(41) NOT NULL,
    amount String NOT NULL,
	duration UInt32 NOT NULL,
	act_type String,
	auto_stake Bool,
	payload Array(UInt8),
	log_index UInt32 NOT NULL
)
ENGINE = ReplacingMergeTree()
PRIMARY KEY (block_height, log_index)
ORDER BY (block_height, log_index)`

type Candidate struct {
	BlockHeight     uint64 `ch:"block_height"`
	Name            string `ch:"name"`
	OperatorAddress string `ch:"operator_address"`
	RewardAddress   string `ch:"reward_address"`
	OwnerAddress    string `ch:"owner_address"`
	CandidateID     string `ch:"candidate_id"`
	Amount          string `ch:"amount"`
	Duration        uint32 `ch:"duration"`
	ActType         string `ch:"act_type"`
	AutoStake       bool   `ch:"auto_stake"`
	Payload         []byte `ch:"payload"`
	LogIndex        uint32 `ch:"log_index"`
}

type Candidates []Candidate

func (Candidate) TableName() string {
	return "candidate"
}

func (Candidate) Columns() []string {
	return []string{
		"block_height",
		"name",
		"operator_address",
		"reward_address",
		"owner_address",
		"candidate_id",
		"amount",
		"duration",
		"act_type",
		"auto_stake",
		"payload",
		"log_index",
	}
}

func (m *Candidate) FetchByName(name string) (*Candidate, error) {
	var err error
	err = db.ChConn().QueryRow(context.Background(), "SELECT * FROM ? WHERE name = ? ORDER BY block_height DESC,log_index DESC LIMIT 1", m.TableName(), name).ScanStruct(m)
	if err != nil {
		return nil, err
	}
	return m, err
}

func (m *Candidate) FetchByOwnerAddressWithHeight(owner string, height uint64) error {
	return db.ChConn().QueryRow(context.Background(), "SELECT * FROM ? WHERE block_height <=? and owner_address = ? ORDER BY block_height DESC,log_index DESC LIMIT 1", m.TableName(), height, owner).ScanStruct(m)
}

func (m *Candidate) FetchByCandidateIDWithHeight(candidateID string, height uint64) error {
	return db.ChConn().QueryRow(context.Background(), "SELECT * FROM ? WHERE block_height <=? and candidate_id = ? ORDER BY block_height DESC,log_index DESC LIMIT 1", m.TableName(), height, candidateID).ScanStruct(m)
}

func (m *Candidate) FetchByNameWithHeight(name string, height uint64) error {
	return db.ChConn().QueryRow(context.Background(), "SELECT * FROM ? WHERE block_height <=? and name = ? ORDER BY block_height DESC,log_index DESC LIMIT 1", m.TableName(), height, name).ScanStruct(m)
}

func GetAllCandidates() (Candidates, error) {
	sql := `SELECT ?
FROM (
    SELECT
        *,
        ROW_NUMBER() OVER (PARTITION BY candidate_id ORDER BY block_height DESC, log_index DESC) AS rn
    FROM
        ?
) AS subquery
WHERE
    rn = 1;`
	var candidates Candidates
	err := db.ChConn().Select(context.Background(), &candidates, sql, strings.Join(Candidate{}.Columns(), ","), Candidate{}.TableName())
	return candidates, err
}

func (m Candidates) ByOwnerAddress(addr string) (Candidate, error) {
	for _, cand := range m {
		if cand.OwnerAddress == addr {
			return cand, nil
		}
	}
	return Candidate{}, errors.New("not found: " + addr)
}

func (m Candidates) ByCandidateID(addr string) (Candidate, error) {
	for _, cand := range m {
		if cand.CandidateID == addr {
			return cand, nil
		}
	}
	return Candidate{}, errors.New("not found: " + addr)
}
