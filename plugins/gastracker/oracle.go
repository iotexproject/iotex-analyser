package main

type gasOracle struct {
	LastBlock       uint64    `json:"LastBlock"`
	SafeGasPrice    float64   `json:"SafeGasPrice"`
	ProposeGasPrice float64   `json:"ProposeGasPrice"`
	FastGasPrice    float64   `json:"FastGasPrice"`
	SuggestBaseFee  float64   `json:"suggestBaseFee"`
	GasUsedRatio    []float64 `json:"gasUsedRatio"`
}
