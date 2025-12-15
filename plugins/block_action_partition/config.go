package main

type Config struct {
	BatchSize int `yaml:"batchSize"`
	// PartitionStep defines the block_height range per partition
	PartitionStep uint64 `yaml:"partitionStep"`
}
