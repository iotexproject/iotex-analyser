package main

// Config controls how aggressively hermes_retention prunes the two giant
// hermes vote tables. Fields left unset (zero value) fall back to the
// built-in defaults; see Start in hermes_retention.go.
type Config struct {
	RetentionEpochs   uint64 `yaml:"retentionEpochs"`   // epochs to keep; default 2185 (~3 months at ~1 epoch/hour)
	TickIntervalHours int    `yaml:"tickIntervalHours"` // purge cadence; default 6
}
