package main

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/kernel"
	slog "github.com/iotexproject/iotex-core/pkg/log"
)

type Network struct {
	StartHeight     uint64
	ContractAddress map[string]string
}

var (
	Networks = map[string]Network{
		"mainnet": {
			StartHeight: 24114600,
			ContractAddress: map[string]string{
				"INSRegistry":             "0xB98825A5cfbFe883c36e2Ac2D4179fc2C92B2bFc",
				"Resolver":                "0x41B9132D4661E016A09a61B314a1DFc0038CE3e8",
				"BaseRegistrar":           "0x41441D51Db3A91d744EE565B693989D0F47B9257",
				"IOTXRegistrarController": "0x8aA6acF9BFeEE0243578305706766065180E68d4",
				"NameWrapper":             "0x26DEFAD6414321b13028A072a46D88400e94abb3",
			},
		},
		"testnet": {
			StartHeight: 20674600,
			ContractAddress: map[string]string{
				"INSRegistry":             "0x845d8ccb0D92174B33AC9A876B65c49Ca4676685",
				"Resolver":                "0x919f2508389c59fe6E896f80c2B70ff53877942B",
				"BaseRegistrar":           "0x40d2FCBE5A396064b69333b12c26CB9F564010CF",
				"IOTXRegistrarController": "0xcFF9867F5ac4b64b613641265262db7dB07Da067",
				"NameWrapper":             "0x92f0926350268a0147E36c4Dfbc4c72Eb11696cF",
			},
		},
	}

	InsDomainContractAddress map[string]string
)

func initAddress() error {
	var err error
	contracts := Networks["mainnet"].ContractAddress
	if kernel.IsTestnet() {
		slog.L().Info("InsDomainContractAddress is testnet")
		contracts = Networks["testnet"].ContractAddress
	}
	InsDomainContractAddress = make(map[string]string, len(contracts))
	for k, v := range contracts {
		addr, err := address.FromHex(v)
		if err != nil {
			return err
		}
		InsDomainContractAddress[k] = addr.String()
	}
	eventABI, err = abi.JSON(strings.NewReader(eventABIJSON))
	return err
}

func isInsDomainContract(addr string) bool {
	for _, contractAddr := range InsDomainContractAddress {
		if addr == contractAddr {
			return true
		}
	}
	return false
}
