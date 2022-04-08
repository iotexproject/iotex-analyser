package main

import (
	"os"
	"testing"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-analyser/models"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestProbationList(t *testing.T) {
	require := require.New(t)
	_, err := initTestConfig()
	require.NoError(err)
	epochNum := uint64(25049)
	chainClient := kernel.ChainClient()
	probationList1, err := models.GetProbationListByEpoch(epochNum)
	require.NoError(err)
	probationList2, err := fetchProbationList(chainClient, epochNum)
	require.NoError(err)
	require.EqualValues(probationList1.ProbationList, probationList2.ProbationList)
	require.Equal(probationList1.IntensityRate, probationList2.IntensityRate)
}

func initTestConfig() (*gorm.DB, error) {
	_, err := config.New(os.Getenv("ConfigPath"))
	if err != nil {
		return nil, err
	}
	return db.Connect()
}
