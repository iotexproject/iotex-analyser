package main

import (
	"io/ioutil"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type AirdripConfig struct {
	Accountant string `yaml:"accountant" env:"AIRDRIP_ACCOUNTANT"`
	Exchange   string `yaml:"exchange" env:"AIRDRIP_EXCHANGE"`
}

func loadAirdripConfig() (cfg *AirdripConfig, err error) {
	if path := getConfigPath(); path != "" {
		body, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read config content")
		}
		if err = yaml.Unmarshal(body, cfg); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal config to struct")
		}
	} else {
		return nil, errors.New("cannot find config file for plugin `airdrip`")
	}

	return
}

var (
	// File names from which we attempt to read configuration.
	DefaultConfigFiles = []string{"airdrip_config.yml", "airdrip_config.yaml"}
)

func getConfigPath() string {
	// TODO (chenzhen): form airdrip config path
	return ""
}
