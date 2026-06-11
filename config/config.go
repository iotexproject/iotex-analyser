package config

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/imdario/mergo"
	"github.com/iotexproject/iotex-core/v2/blockchain/genesis"
	coredb "github.com/iotexproject/iotex-core/v2/db"
	"github.com/iotexproject/iotex-core/v2/pkg/log"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/sethvargo/go-envconfig"
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
			BatchSize:          64,
			MaxCallRecvMsgSize: 1024 * 1024 * 4, // 4MB
		},
		BlockDB: Blockdao{
			Config: coredb.Config{
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
			BatchSize: 512,
		},
		SubLogs: make(map[string]log.GlobalConfig),
		Genesis: genesis.Default,
	}
)

type (
	Server struct {
		Http          string         `yaml:"http"`
		Addr          string         `yaml:"addr"`
		HTTPAdminPort int            `yaml:"httpAdminPort"`
		Plugins       []string       `yaml:"plugins"`
		PluginConfigs map[string]any `yaml:"pluginConfigs"`
	}
	Database struct {
		Driver   string `yaml:"driver" env:"DB_DRIVER"`
		Host     string `yaml:"host" env:"DB_HOST"`
		Port     string `yaml:"port" env:"DB_PORT"`
		User     string `yaml:"user"  env:"DB_USER"`
		Password string `yaml:"password"  env:"DB_PASSWORD"`
		Name     string `yaml:"name"  env:"DB_NAME"`
		Debug    bool   `yaml:"debug"  env:"DB_DEBUG"`
	}
	Iotex struct {
		EVMNetworkID       uint32   `yaml:"evmNetworkID"`
		CrawlMode          bool     `yaml:"crawlMode"`
		CrawlHeight        []uint64 `yaml:"crawlHeight"`
		DisableRebuildDB   bool     `yaml:"disableRebuildDB"`
		CatchUpMode        bool     `yaml:"catchUpMode"`
		CatchUpStartHeight uint64   `yaml:"catchUpStartHeight"`
		// CatchUpAllowPlugins is an operator override: plugins listed here
		// are loaded in catch-up mode even if they have not declared
		// themselves catch-up safe via plugin.CatchUpAdapter. Plugins not
		// listed AND not declared safe are skipped with a warning.
		CatchUpAllowPlugins []string `yaml:"catchUpAllowPlugins"`
		ChainEndPoint      string   `yaml:"chainEndPoint" env:"IOTEX_CHAIN_END_POINT"`
		ChainInsecure      bool     `yaml:"chainInsecure"`
		// EthArchiveEndPoint is an eth-json-rpc endpoint backed by an archive node.
		// Used for state-at-height queries (eth_getCode, eth_getTransactionCount).
		EthArchiveEndPoint string `yaml:"ethArchiveEndPoint" env:"IOTEX_ETH_ARCHIVE_END_POINT"`
		BatchSize          uint64 `yaml:"batchSize"`          //default 64, ~ 10 blocks
		MaxCallRecvMsgSize int    `yaml:"maxCallRecvMsgSize"` //default 4MB, in bytes
	}
	Config struct {
		Genesis  genesis.Genesis             `yaml:"genesis"`
		Server   Server                      `yaml:"server"`
		Database Database                    `yaml:"database"`
		Iotex    Iotex                       `yaml:"iotex"`
		BlockDB  Blockdao                    `yaml:"blockDB"`
		Log      log.GlobalConfig            `yaml:"log" json:"-"`
		SubLogs  map[string]log.GlobalConfig `yaml:"subLogs" json:"-"`
	}
	Blockdao struct {
		coredb.Config `yaml:",inline"`
		BatchSize     int `yaml:"batchSize"`
	}
)

func New(path string) (cfg *Config, err error) {
	body, err := ioutil.ReadFile(path)
	if err != nil {
		return cfg, errors.Wrap(err, "failed to read config content")
	}
	cfg = &Default
	var envCfg Config
	if err := envconfig.Process(context.Background(), &envCfg); err != nil {
		return cfg, errors.Wrap(err, "failed to process envconfig to struct")
	}
	if err = yaml.Unmarshal(body, cfg); err != nil {
		return cfg, errors.Wrap(err, "failed to unmarshal config to struct")
	}
	if err := mergo.Merge(&Default.Database, envCfg.Database, mergo.WithOverride); err != nil {
		return cfg, errors.Wrap(err, "failed to merge config")
	}
	return
}

var (
	// File names from which we attempt to read configuration.
	DefaultConfigFiles = []string{"config.yml", "config.yaml"}

	// Launchd doesn't set root env variables, so there is default
	DefaultConfigDirs = []string{getCurrentDirectory(), "~/.iotex-analyser", "/usr/local/etc/iotex-analyser", "/etc/iotex-analyser"}
)

func getCurrentDirectory() string {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return ""
	}
	return strings.Replace(dir, "\\", "/", -1)
}

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

var (
	evmNetworkID uint32
)

func SetEVMNetworkID(id uint32) {
	atomic.StoreUint32(&evmNetworkID, id)
}

func EVMNetworkID() uint32 {
	id := atomic.LoadUint32(&evmNetworkID)
	if id == 0 {
		panic("EVM network ID is not set")
	}
	return id
}
