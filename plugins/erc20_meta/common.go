package main

import (
	"context"
	"encoding/hex"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/iotexproject/iotex-core/v2/action"
	"github.com/iotexproject/iotex-core/v2/test/identityset"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
)

const (
	ERC20ABI = "[{\"inputs\":[{\"internalType\":\"string\",\"name\":\"name_\",\"type\":\"string\"},{\"internalType\":\"string\",\"name\":\"symbol_\",\"type\":\"string\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Approval\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"dst\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"wad\",\"type\":\"uint256\"}],\"name\":\"Deposit\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"}],\"name\":\"Transfer\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"src\",\"type\":\"address\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"wad\",\"type\":\"uint256\"}],\"name\":\"Withdrawal\",\"type\":\"event\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"owner\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"}],\"name\":\"allowance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"approve\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"account\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"decimals\",\"outputs\":[{\"internalType\":\"uint8\",\"name\":\"\",\"type\":\"uint8\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"subtractedValue\",\"type\":\"uint256\"}],\"name\":\"decreaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"spender\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"addedValue\",\"type\":\"uint256\"}],\"name\":\"increaseAllowance\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"symbol\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transfer\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"from\",\"type\":\"address\"},{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint256\",\"name\":\"amount\",\"type\":\"uint256\"}],\"name\":\"transferFrom\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"
)

var erc20ABI abi.ABI

func initErc20() error {
	var err error
	erc20ABI, err = abi.JSON(strings.NewReader(ERC20ABI))
	return err
}

// ReadERC20Decimals read ERC20 decimals
func ReadERC20Decimals(client iotexapi.APIServiceClient, contractAddr string) (int, error) {
	decimals := 0 //default decimal

	transferAmount := big.NewInt(0)
	callerAddress := identityset.Address(30).String()
	callData, _ := hex.DecodeString("313ce567")
	execution := action.NewExecution(contractAddr, transferAmount, callData)
	request := &iotexapi.ReadContractRequest{
		Execution:     execution.Proto(),
		CallerAddress: callerAddress,
	}

	res, err := client.ReadContract(context.Background(), request)
	if err != nil {
		return decimals, err
	}
	if res.Data != "" {
		tmp, _ := strconv.ParseInt(res.Data, 16, 64)
		decimals = int(tmp)
	}
	return decimals, nil
}

func ReadERC20Symbol(client iotexapi.APIServiceClient, contractAddr string) (string, error) {
	symbol := "" //default symbol

	transferAmount := big.NewInt(0)
	callerAddress := identityset.Address(30).String()
	callData, _ := hex.DecodeString("95d89b41")
	execution := action.NewExecution(contractAddr, transferAmount, callData)
	request := &iotexapi.ReadContractRequest{
		Execution:     execution.Proto(),
		CallerAddress: callerAddress,
	}

	res, err := client.ReadContract(context.Background(), request)
	if err != nil {
		return symbol, nil
	}
	if res.Data != "" {
		data, err := hex.DecodeString(res.Data)
		if err != nil {
			return symbol, err
		}
		unPack, err := erc20ABI.Unpack("symbol", data)
		if err != nil {
			//if can't unpack, return empty string
			return "", nil
		}
		return unPack[0].(string), nil
	}
	return symbol, nil
}

func ReadERC20Name(client iotexapi.APIServiceClient, contractAddr string) (string, error) {
	name := "" //default name

	transferAmount := big.NewInt(0)
	callerAddress := identityset.Address(30).String()
	callData, _ := hex.DecodeString("06fdde03")
	execution := action.NewExecution(contractAddr, transferAmount, callData)
	request := &iotexapi.ReadContractRequest{
		Execution:     execution.Proto(),
		CallerAddress: callerAddress,
	}

	res, err := client.ReadContract(context.Background(), request)
	if err != nil {
		return name, nil
	}
	if res.Data != "" {
		data, err := hex.DecodeString(res.Data)
		if err != nil {
			return name, err
		}
		unPack, err := erc20ABI.Unpack("name", data)
		if err != nil {
			//if can't unpack, return empty string
			return "", nil
		}
		return unPack[0].(string), nil
	}
	return name, nil
}
