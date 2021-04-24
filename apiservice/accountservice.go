package apiservice

import (
	"context"
	"database/sql"
	"math/big"

	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/api"
	"github.com/iotexproject/iotex-analyser/kernel"
	"github.com/iotexproject/iotex-core/ioctl/util"
)

type AccountService struct {
	api.UnimplementedAccountServiceServer
}

/*
mysql> SELECT SUM(amount),SUM(gas_price*gas_consumed) FROM block_action where block_height<=8844615 and `from`='io1fuhhg9jgdxwpms9dsdfwjdc90nt7v67hx40cd8';
+----------------------+-----------------------------+
| SUM(amount)          | SUM(gas_price*gas_consumed) |
+----------------------+-----------------------------+
| 33000000000000000000 |        24801116000000000000 |
+----------------------+-----------------------------+
1 row in set (0.02 sec)
mysql> SELECT SUM(amount) FROM block_action where block_height<=8844615 and `to`='io1fuhhg9jgdxwpms9dsdfwjdc90nt7v67hx40cd8';
+------------------------+
| SUM(amount)            |
+------------------------+
| 1020000000000000000000 |
+------------------------+
1 row in set (0.01 sec)
*/
func (s *AccountService) GetIotexBalanceByHeight(ctx context.Context, req *api.AccountRequest) (*api.AccountResponse, error) {
	resp := &api.AccountResponse{}
	addr := req.GetAddress()
	height := req.GetHeight()

	if addr[:2] == "0x" || addr[:2] == "0X" {
		add, err := address.FromHex(addr)
		if err != nil {
			return nil, err
		}

		addr = add.String()
	}

	db := kernel.GetDB()

	//get receive amount
	var toAmount sql.NullString
	query := "SELECT SUM(amount) FROM block_action WHERE block_height<=? AND `to`=?"
	err := db.QueryRow(query, height, addr).Scan(&toAmount)
	if err != nil {
		return nil, err
	}

	//get cost amount
	var fromAmount, gasFee sql.NullString
	query = "SELECT SUM(amount),SUM(gas_price*gas_consumed) FROM block_action WHERE block_height<=? AND `from`=?"
	err = db.QueryRow(query, height, addr).Scan(&fromAmount, &gasFee)
	if err != nil {
		return nil, err
	}

	to, ok := big.NewInt(0).SetString(toAmount.String, 10)
	if !ok {
		to = big.NewInt(0)
	}
	from, ok := big.NewInt(0).SetString(fromAmount.String, 10)
	if !ok {
		from = big.NewInt(0)
	}
	gas, _ := big.NewInt(0).SetString(gasFee.String, 10)
	if !ok {
		gas = big.NewInt(0)
	}
	// to, _ = big.NewInt(0).SetString("1020000000000000000000", 10)
	// from, _ = big.NewInt(0).SetString("33000000000000000000", 10)
	// gas, _ = big.NewInt(0).SetString("24801116000000000000", 10)
	balance := new(big.Int).Sub(to, from)
	balance = new(big.Int).Sub(balance, gas)

	resp.Balance = util.RauToString(balance, util.IotxDecimalNum)
	return resp, nil
}

//grpcurl -plaintext -d '{"address": "io1ryztljunahyml9s7atfwtsx7s8wvr5maufa6zp", "height":8927781 }' 127.0.0.1:7777 api.AccountService.GetErc20TokenBalanceByHeight
//curl -d '{"address": "io1ryztljunahyml9s7atfwtsx7s8wvr5maufa6zp", "height":8927781 }' http://rvice.GetErc20TokenBalanceByHeight
func (s *AccountService) GetErc20TokenBalanceByHeight(ctx context.Context, req *api.AccountErc20TokenRequest) (*api.AccountResponse, error) {
	resp := &api.AccountResponse{}
	addr := req.GetAddress()
	height := req.GetHeight()
	contractAddress := req.GetContractAddress()

	if addr[:2] == "0x" || addr[:2] == "0X" {
		add, err := address.FromHex(addr)
		if err != nil {
			return nil, err
		}

		addr = add.String()
	}

	if contractAddress[:2] == "0x" || contractAddress[:2] == "0X" {
		add, err := address.FromHex(contractAddress)
		if err != nil {
			return nil, err
		}

		contractAddress = add.String()
	}

	db := kernel.GetDB()

	//get receive amount
	var toAmount sql.NullString
	query := "SELECT SUM(amount) FROM token_erc20 WHERE block_height<=? AND `to`=? AND `contract_address`=?"
	err := db.QueryRow(query, height, addr, contractAddress).Scan(&toAmount)
	if err != nil {
		return nil, err
	}

	//get cost amount
	var fromAmount sql.NullString
	query = "SELECT SUM(amount) FROM token_erc20 WHERE block_height<=? AND `from`=? AND `contract_address`=?"
	err = db.QueryRow(query, height, addr, contractAddress).Scan(&fromAmount)
	if err != nil {
		return nil, err
	}

	to, ok := big.NewInt(0).SetString(toAmount.String, 10)
	if !ok {
		to = big.NewInt(0)
	}
	from, ok := big.NewInt(0).SetString(fromAmount.String, 10)
	if !ok {
		from = big.NewInt(0)
	}

	balance := new(big.Int).Sub(to, from)

	resp.Balance = util.RauToString(balance, 6)
	return resp, nil
}
