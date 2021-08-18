package main

import (
	"context"
	"encoding/hex"
	"math/big"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-antenna-go/v2/account"
	"github.com/iotexproject/iotex-antenna-go/v2/iotex"
	"github.com/iotexproject/iotex-proto/golang/iotexapi"
	"github.com/pkg/errors"
)

// ioAddrToEvmAddr converts IoTeX address into evm address
func ioAddrToEvmAddr(ioAddr string) (common.Address, error) {

	address, err := address.FromString(ioAddr)
	if err != nil {
		return common.Address{}, err
	}
	return common.BytesToAddress(address.Bytes()), nil
}

// stringToBytes32 converts string to bytes32
func stringToBytes32(addr string) [32]byte {
	var name [32]byte
	copy(name[:], addr)
	return name
}

func commitContract(users [][32]byte, shares []*big.Int) error {
	account, err := account.HexStringToAccount(Default.Airdrip.PrivateKey)
	if err != nil {
		return err
	}
	endpoint := Default.Iotex.ChainEndPoint
	conn, err := iotex.NewDefaultGRPCConn(endpoint)
	if err != nil {
		return err
	}
	defer conn.Close()

	c := iotex.NewAuthedClient(iotexapi.NewAPIServiceClient(conn), account)

	cstring := Default.Airdrip.ContractAddress
	caddr, err := address.FromString(cstring)
	if err != nil {
		return err
	}

	ctx := context.Background()
	airdripABI, err := abi.JSON(strings.NewReader(AirdripABI))
	if err != nil {
		return err
	}

	gasPriceStr := Default.Airdrip.GasPrice
	gasPrice, ok := big.NewInt(0).SetString(gasPriceStr, 10)
	if !ok {
		return errors.New("failed to convert string to big int")
	}
	gasLimitStr := Default.Airdrip.GasLimit
	gasLimit, err := strconv.Atoi(gasLimitStr)
	if err != nil {
		return err
	}
	h, err := c.Contract(caddr, airdripABI).Execute("updateShares", users, shares).
		SetGasPrice(gasPrice).SetGasLimit(uint64(gasLimit)).Call(ctx)
	if err != nil {
		return err
	}

	resp, err := c.API().GetReceiptByAction(ctx, &iotexapi.GetReceiptByActionRequest{
		ActionHash: hex.EncodeToString(h[:]),
	})
	if err != nil {
		return err
	}
	if resp.ReceiptInfo.Receipt.Status != 1 {
		return errors.Errorf("commitContract failed: %x", h)
	}

	return nil
}
