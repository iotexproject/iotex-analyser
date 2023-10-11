package main

import (
	"context"
	"encoding/hex"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-core/action"
	slog "github.com/iotexproject/iotex-core/pkg/log"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	eventABIJSON = `[
		{
			"anonymous": false,
			"inputs": [
			  {
				"indexed": true,
				"internalType": "bytes32",
				"name": "node",
				"type": "bytes32"
			  },
			  {
				"indexed": false,
				"internalType": "string",
				"name": "name",
				"type": "string"
			  }
			],
			"name": "NameChanged",
			"type": "event"
		  },
		  {
			"anonymous": false,
			"inputs": [
			  {
				"indexed": true,
				"internalType": "bytes32",
				"name": "node",
				"type": "bytes32"
			  },
			  {
				"indexed": false,
				"internalType": "address",
				"name": "a",
				"type": "address"
			  }
			],
			"name": "AddrChanged",
			"type": "event"
		  },
		  {
			"anonymous": false,
			"inputs": [
			  {
				"indexed": true,
				"internalType": "uint256",
				"name": "id",
				"type": "uint256"
			  },
			  {
				"indexed": true,
				"internalType": "address",
				"name": "owner",
				"type": "address"
			  },
			  {
				"indexed": false,
				"internalType": "uint256",
				"name": "expires",
				"type": "uint256"
			  }
			],
			"name": "NameRegistered",
			"type": "event"
		  },
		  {
			"anonymous": false,
			"inputs": [
			  {
				"indexed": true,
				"internalType": "uint256",
				"name": "id",
				"type": "uint256"
			  },
			  {
				"indexed": false,
				"internalType": "uint256",
				"name": "expires",
				"type": "uint256"
			  }
			],
			"name": "NameRenewed",
			"type": "event"
		  },
		  {
			"anonymous": false,
			"inputs": [
			  {
				"indexed": true,
				"internalType": "address",
				"name": "from",
				"type": "address"
			  },
			  {
				"indexed": true,
				"internalType": "address",
				"name": "to",
				"type": "address"
			  },
			  {
				"indexed": true,
				"internalType": "uint256",
				"name": "tokenId",
				"type": "uint256"
			  }
			],
			"name": "Transfer",
			"type": "event"
		  },
		  {
			"anonymous": false,
			"inputs": [
			  {
				"indexed": true,
				"internalType": "bytes32",
				"name": "node",
				"type": "bytes32"
			  },
			  {
				"indexed": false,
				"internalType": "address",
				"name": "resolver",
				"type": "address"
			  }
			],
			"name": "NewResolver",
			"type": "event"
		  },
		  {
			"anonymous": false,
			"inputs": [
			  {
				"indexed": true,
				"internalType": "bytes32",
				"name": "node",
				"type": "bytes32"
			  },
			  {
				"indexed": false,
				"internalType": "bytes",
				"name": "name",
				"type": "bytes"
			  },
			  {
				"indexed": false,
				"internalType": "address",
				"name": "owner",
				"type": "address"
			  },
			  {
				"indexed": false,
				"internalType": "uint32",
				"name": "fuses",
				"type": "uint32"
			  },
			  {
				"indexed": false,
				"internalType": "uint64",
				"name": "expiry",
				"type": "uint64"
			  }
			],
			"name": "NameWrapped",
			"type": "event"
		  }
	]
	`
	eventABI abi.ABI
)

func handleInsRegistry(ctx context.Context, tx *gorm.DB, alog *action.Log, actionHash string) error {
	thash := common.BytesToHash(alog.Topics[0][:])
	switch thash {
	case eventABI.Events["NewResolver"].ID:
		event := struct {
			Node     common.Hash
			Resolver common.Address
		}{}
		err := kernel.UnpackLog(eventABI, &event, "NewResolver", alog)
		if err != nil {
			return err
		}
		slog.L().Info("handleInsRegistry NewResolver", zap.String("node", event.Node.String()), zap.String("resolver", event.Resolver.String()))
	}
	return nil
}

func handleResolver(ctx context.Context, tx *gorm.DB, alog *action.Log, actionHash string) error {
	thash := common.BytesToHash(alog.Topics[0][:])
	switch thash {
	case eventABI.Events["NameChanged"].ID:

		node := hex.EncodeToString(alog.Topics[1][:])
		event := struct {
			Node common.Hash
			Name string
		}{}
		err := kernel.UnpackLog(eventABI, &event, "NameChanged", alog)
		if err != nil {
			return err
		}
		slog.L().Info("handleResolver NameChanged", zap.String("node", node), zap.String("name", event.Name))
	case eventABI.Events["AddrChanged"].ID:
		event := struct {
			Node common.Hash
			A    common.Address
		}{}
		err := kernel.UnpackLog(eventABI, &event, "AddrChanged", alog)
		if err != nil {
			return err
		}
		node := hex.EncodeToString(alog.Topics[1][:])

		slog.L().Info("handleResolver AddrChanged", zap.String("node", node), zap.String("address", event.A.String()))
	}
	return nil
}

func handleBaseRegistrar(ctx context.Context, tx *gorm.DB, alog *action.Log, actionHash string) error {
	thash := common.BytesToHash(alog.Topics[0][:])
	switch thash {
	case eventABI.Events["NameRegistered"].ID:
		event := struct {
			Id      *big.Int
			Owner   common.Address
			Expires *big.Int
		}{}
		err := kernel.UnpackLog(eventABI, &event, "NameRegistered", alog)
		if err != nil {
			return err
		}
		slog.L().Info("handleBaseRegistrar NameRegistered", zap.String("id", event.Id.String()), zap.String("owner", event.Owner.String()), zap.String("expires", event.Expires.String()))
	case eventABI.Events["NameRenewed"].ID:
		event := struct {
			Id      *big.Int
			Expires *big.Int
		}{}
		err := kernel.UnpackLog(eventABI, &event, "NameRenewed", alog)
		if err != nil {
			return err
		}
		slog.L().Info("handleBaseRegistrar NameRenewed", zap.String("id", event.Id.String()), zap.String("expires", event.Expires.String()))
	case eventABI.Events["Transfer"].ID:
		event := struct {
			From    common.Address
			To      common.Address
			TokenId *big.Int
		}{}
		err := kernel.UnpackLog(eventABI, &event, "Transfer", alog)
		if err != nil {
			return err
		}
		slog.L().Info("handleBaseRegistrar Transfer", zap.String("from", event.From.String()), zap.String("to", event.To.String()), zap.String("tokenId", event.TokenId.String()))

	}
	return nil
}
func handleNameWrapper(ctx context.Context, tx *gorm.DB, alog *action.Log, actionHash string) error {
	thash := common.BytesToHash(alog.Topics[0][:])
	switch thash {
	case eventABI.Events["NameWrapped"].ID:
		event := struct {
			Node   common.Hash
			Name   []byte
			Owner  common.Address
			Fuses  uint32
			Expiry uint64
		}{}
		err := kernel.UnpackLog(eventABI, &event, "NameWrapped", alog)
		if err != nil {
			return err
		}
		slog.L().Info("handleNameWrapper NameWrapped", zap.String("node", event.Node.String()), zap.String("name", string(event.Name)), zap.String("owner", event.Owner.String()), zap.String("fuses", string(event.Fuses)), zap.Uint64("expiry", event.Expiry))
	}
	return nil
}
