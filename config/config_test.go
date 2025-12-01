package config

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

func TestConfigDeserialize(t *testing.T) {
	configYAML := `
blockDB:
  dbPath: "/path/to/db"
  batchSize: 8000
`
	var cfg Config
	err := yaml.Unmarshal([]byte(configYAML), &cfg)
	require.NoError(t, err)
	require.Equal(t, "/path/to/db", cfg.BlockDB.DbPath)
	require.Equal(t, 8000, cfg.BlockDB.BatchSize)
}
