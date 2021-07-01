package apiservice

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
	}
)

type (
	Config struct {
		Genesis `yaml:"genesis"`
	}
	Genesis struct {
		VoteWeightCalConsts genesis.VoteWeightCalConsts `yaml:"voteWeightCalConsts"`
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
