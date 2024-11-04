package kernel

import (
	"context"
	"encoding/hex"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/test/identityset"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/pkg/errors"
)

var (
	//https://eips.ethereum.org/EIPS/eip-721
	ERC721InterfaceID = [4]byte{0x80, 0xac, 0x58, 0xcd}
	//https://eips.ethereum.org/EIPS/eip-1155
	ERC1155InterfaceID = [4]byte{0xd9, 0xb6, 0x7a, 0x26}
	SBTInterfaceID     = [4]byte{180, 90, 60, 14}
	successStatus      = uint64(1)
	nonce              = uint64(1)
	transferAmount     = big.NewInt(0)
	gasLimit           = uint64(100000)
	gasPrice           = big.NewInt(10000000)
	callerAddress      = identityset.Address(30).String()
)

// UnpackLog parses the log data and returns the corresponding event name and arguments
func UnpackLog(a abi.ABI, out interface{}, event string, log *action.Log) error {
	if len(log.Data) > 0 {
		if err := a.UnpackIntoInterface(out, event, log.Data); err != nil {
			return errors.Wrap(err, "failed to unpack log")
		}
	}
	var indexed abi.Arguments
	for _, arg := range a.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	topics := []common.Hash{}
	for _, topic := range log.Topics {
		topics = append(topics, common.BytesToHash(topic[:]))
	}
	return abi.ParseTopics(out, indexed, topics[1:])
}

func IsErc721(addr string) (bool, error) {
	return CheckInterfaceID(addr, ERC721InterfaceID)
}

func IsErc1155(addr string) (bool, error) {
	return CheckInterfaceID(addr, ERC1155InterfaceID)
}

func IsSBT(addr string) (bool, error) {
	return CheckInterfaceID(addr, SBTInterfaceID)
}

func CheckInterfaceID(addr string, interfaceID [4]byte) (bool, error) {
	cli := ChainClient()
	callData := common.FromHex("0x01ffc9a70000000000000000000000000000000000000000000000000000000000000000")
	copy(callData[4:], interfaceID[:])
	execution, err := action.NewExecution(addr, nonce, transferAmount, gasLimit, gasPrice, callData)
	if err != nil {
		return false, err
	}
	request := &iotexapi.ReadContractRequest{
		Execution:     execution.Proto(),
		CallerAddress: callerAddress,
	}
	res, err := cli.ReadContract(context.Background(), request)
	if err != nil {
		return false, err
	}
	if res.Receipt.Status == successStatus {
		if len(res.Data) == 0 {
			return false, nil
		}
		data, err := hex.DecodeString(res.Data)
		if err != nil {
			return false, err
		}
		// supportsInterface(bytes4 interfaceID) should return bool
		if len(data) != 32 {
			return false, nil
		}

		return data[31] == byte(1), nil
	}
	return false, nil
}
