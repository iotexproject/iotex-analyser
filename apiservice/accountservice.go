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

//curl -d '{"request":[{"address": "io1ryztljunahyml9s7atfwtsx7s8wvr5maufa6zp", "height":8927781 }]}' http://127.0.0.1:7778/api.AccountService.GetIotexBalanceByHeight
func (s *AccountService) GetIotexBalanceByHeight(ctx context.Context, req *api.AccountRequest) (*api.AccountResponse, error) {
	resp := &api.AccountResponse{}
	db := kernel.GetDB()
	for _, req := range req.GetRequest() {
		addr := req.GetAddress()
		height := req.GetHeight()
		if addr[:2] == "0x" || addr[:2] == "0X" {
			add, err := address.FromHex(addr)
			if err != nil {
				return nil, err
			}

			addr = add.String()
		}
		var amount sql.NullString
		query := "SELECT sum(in_flow)-sum(out_flow) from account_income WHERE block_height<=? and account_address=?"
		err := db.QueryRow(query, height, addr).Scan(&amount)
		if err != nil {
			return nil, err
		}
		balance, ok := big.NewInt(0).SetString(amount.String, 10)
		if !ok {
			balance = big.NewInt(0)
		}
		res := &api.BalanceResponse{
			Address: req.GetAddress(),
			Height:  req.GetHeight(),
			Balance: util.RauToString(balance, util.IotxDecimalNum),
		}
		resp.Response = append(resp.Response, res)
	}

	return resp, nil
}

//grpcurl -plaintext -d '{"address": "io1ryztljunahyml9s7atfwtsx7s8wvr5maufa6zp", "height":8927781 }' 127.0.0.1:7777 api.AccountService.GetErc20TokenBalanceByHeight
//curl -d '{"request":[{"address": "io1ryztljunahyml9s7atfwtsx7s8wvr5maufa6zp", "contract_address": "io1ryztljunahyml9s7atfwtsx7s8wvr5maufa6zp", "height":8927781 }]}' http://127.0.0.1:7778/api.AccountService.GetErc20TokenBalanceByHeight
func (s *AccountService) GetErc20TokenBalanceByHeight(ctx context.Context, req *api.AccountErc20TokenRequest) (*api.AccountResponse, error) {
	resp := &api.AccountResponse{}
	db := kernel.GetDB()
	for _, req := range req.GetRequest() {
		addr := req.GetAddress()
		height := req.GetHeight()
		contractAddress := req.GetContractAddress()
		if len(addr) > 2 && (addr[:2] == "0x" || addr[:2] == "0X") {
			add, err := address.FromHex(addr)
			if err != nil {
				return nil, err
			}

			addr = add.String()
		}

		if len(contractAddress) > 2 && (contractAddress[:2] == "0x" || contractAddress[:2] == "0X") {
			add, err := address.FromHex(contractAddress)
			if err != nil {
				return nil, err
			}

			contractAddress = add.String()
		}
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
		res := &api.BalanceResponse{
			Address:         req.GetAddress(),
			ContractAddress: req.GetContractAddress(),
			Height:          req.GetHeight(),
			Balance:         util.RauToString(balance, 6),
		}
		resp.Response = append(resp.Response, res)
	}

	return resp, nil
}
