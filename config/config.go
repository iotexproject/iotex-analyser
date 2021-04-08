package config

import (
	"io/ioutil"
	"os"
	"path/filepath"

	coreconfig "github.com/iotexproject/iotex-core/config"
	"github.com/iotexproject/iotex-core/pkg/log"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

var (
	// Default is the default config
	Default = Config{
		Server: Server{
			Http: "",
			Addr: "/tmp/iotex-analyser.sock",
		},
		Iotex: Iotex{
			BatchSize: 64,
		},
		BlockDB: coreconfig.DB{
			NumRetries:            3,
			MaxCacheSize:          64,
			BlockStoreBatchSize:   16,
			V2BlocksToSplitDB:     1000000,
			Compressor:            "Snappy",
			CompressLegacy:        false,
			SplitDBSizeMB:         0,
			SplitDBHeight:         900000,
			HistoryStateRetention: 2000,
		},
		SubLogs: make(map[string]log.GlobalConfig),
	}
)

type (
	Server struct {
		Http          string   `yaml:"http"`
		Addr          string   `yaml:"addr"`
		HTTPAdminPort int      `yaml:"httpAdminPort"`
		Plugins       []string `yaml:"plugins"`
	}
	Database struct {
		Dsn string `yaml:"dsn"`
	}
	Iotex struct {
		CatchUpMode        bool   `yaml:"catchUpMode"`
		CatchUpStartHeight uint64 `yaml:"catchUpStartHeight"`
		ChainEndPoint      string `yaml:"chainEndPoint"`
		ChainInsecure      bool   `yaml:"chainInsecure"`
		BatchSize          uint64 `yaml:"batchSize"` //default 64, ~ 10 blocks
	}
	Config struct {
		Server   Server                      `yaml:"server"`
		Database Database                    `yaml:"database"`
		Iotex    Iotex                       `yaml:"iotex"`
		BlockDB  coreconfig.DB               `yaml:"blockDB"`
		Log      log.GlobalConfig            `yaml:"log" json:"-"`
		SubLogs  map[string]log.GlobalConfig `yaml:"subLogs" json:"-"`
	}
)

func New(path string) (cfg *Config, err error) {
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

var (
	// File names from which we attempt to read configuration.
	DefaultConfigFiles = []string{"config.yml", "config.yaml"}

	// Launchd doesn't set root env variables, so there is default
	DefaultConfigDirs = []string{"~/.iotex-analyser", "/usr/local/etc/iotex-analyser", "/etc/iotex-analyser"}
)

// FileExists checks to see if a file exist at the provided path.
func FileExists(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			// ignore missing files
			return false, nil
		}
		return false, err
	}
	f.Close()
	return true, nil
}

// FindDefaultConfigPath returns the first path that contains a config file.
// If none of the combination of DefaultConfigDirs and DefaultConfigFiles
// contains a config file, return empty string.
func FindDefaultConfigPath() string {
	for _, configDir := range DefaultConfigDirs {
		for _, configFile := range DefaultConfigFiles {
			dirPath, err := homedir.Expand(configDir)
			if err != nil {
				continue
			}
			path := filepath.Join(dirPath, configFile)
			if ok, _ := FileExists(path); ok {
				return path
			}
		}
	}
	return ""
}
