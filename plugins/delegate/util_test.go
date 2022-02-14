package main

import (
	"testing"

	"github.com/iotexproject/iotex-analyser/config"
	"github.com/iotexproject/iotex-analyser/db"
	"github.com/stretchr/testify/require"
)

func Test_getDelegateActive(t *testing.T) {
	require := require.New(t)
	config.Default.Database = config.Database{
		Driver:   "postgres",
		Name:     "mainlive",
		Host:     "127.0.0.1",
		Port:     "5432",
		User:     "postgres",
		Password: "admin",
		Debug:    true,
	}
	_, err := db.Connect()
	require.NoError(err)
	getDelegateActive(4)
}
