package main

import (
	"io/ioutil"

	"github.com/iotexproject/iotex-core/blockchain/genesis"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

var (
	// Default config
	Default = Config{
		Genesis: Genesis{
			VoteWeightCalConsts: genesis.VoteWeightCalConsts{
				DurationLg: 1.2,
				AutoStake:  1,
				SelfStake:  1.06,
			},
		},
		Iotex: Iotex{
			ChainEndPoint: "",
		},
		Airdrip: Airdrip{
			InitHeight:      0,
			ContractAddress: "",
			GasPrice:        "",
			GasLimit:        "",
			PrivateKey:      "",
		},
	}
)

type (
	Config struct {
		Airdrip Airdrip `yaml:"airdrip"`
		Genesis `yaml:"genesis"`
		Iotex   `yaml:"iotex"`
	}
	Genesis struct {
		VoteWeightCalConsts genesis.VoteWeightCalConsts `yaml:"voteWeightCalConsts"`
	}
	Iotex struct {
		ChainEndPoint string `yaml:"chainEndPoint"`
	}
	Airdrip struct {
		InitHeight      uint64 `yaml:"initHeight"`
		ContractAddress string `yaml:"contractAddress"`
		GasPrice        string `yaml:"gasPrice"`
		GasLimit        string `yaml:"gasLimit"`
		PrivateKey      string `yaml:"privateKey"`
	}
)

func newConfig(path string) (cfg *Config, err error) {
	body, err := ioutil.ReadFile(path)
	if err != nil {
		return cfg, errors.Wrap(err, "failed to read config content")
	}
	cfg = &Default
	if err = yaml.Unmarshal(body, cfg); err != nil {
		return cfg, errors.Wrap(err, "failed to unmarshal config to struct")
	}
	return
}
