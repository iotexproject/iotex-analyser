package main

import (
	"context"
	"io/ioutil"
	"path/filepath"

	"github.com/imdario/mergo"
	"github.com/iotexproject/iotex-analyser/config"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/sethvargo/go-envconfig"
	"gopkg.in/yaml.v2"
)

var (
	// Default config
	Default = AirdripConfig{}
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
		cfg = &Default
		var envCfg AirdripConfig
		if err := envconfig.Process(context.Background(), &envCfg); err != nil {
			return cfg, errors.Wrap(err, "failed to process envconfig to struct")
		}
		if err = yaml.Unmarshal(body, cfg); err != nil {
			return cfg, errors.Wrap(err, "failed to unmarshal config to struct")
		}
		if err := mergo.Merge(cfg, envCfg, mergo.WithOverride); err != nil {
			return cfg, errors.Wrap(err, "failed to merge config")
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
	for _, configDir := range config.DefaultConfigDirs {
		for _, configFile := range DefaultConfigFiles {
			dirPath, err := homedir.Expand(configDir)
			if err != nil {
				continue
			}
			path := filepath.Join(dirPath, configFile)
			if ok, _ := config.FileExists(path); ok {
				return path
			}
		}
	}

	return ""
}
