package main

import (
	"io/ioutil"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

var (
	// Default config
	Default = Config{
		Genesis: Genesis{
			Account: Account{
				InitBalanceMap: make(map[string]string),
			},
		},
	}
)

type (
	Config struct {
		Genesis `yaml:"genesis"`
	}
	Genesis struct {
		Account `yaml:"account"`
	}
	// Account contains the configs for account protocol
	Account struct {
		// InitBalanceMap is the address and initial balance mapping before the first block.
		InitBalanceMap map[string]string `yaml:"initBalances"`
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
