package apiservice

import (
	"context"

	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/api"
	"github.com/iotexproject/iotex-analyser/db"
)

type XrcType int

const (
	Xrc20 XrcType = iota
	Xrc721
)

type ActionsService struct {
	api.UnimplementedActionsServiceServer
}

//curl -d '{"address": "io102s4660k3cynae2r8gde6scg74mf6f7k9dq955", "height":11900487 }' http://127.0.0.1:7778/api.ActionsService.GetActionsByAddress
func (s *ActionsService) GetActionsByAddress(ctx context.Context, req *api.ActionsRequest) (*api.ActionsByAddressResponse, error) {
	resp := &api.ActionsByAddressResponse{
		Count: 0,
	}
	db := db.DB()
	addr := req.GetAddress()
	if addr[:2] == "0x" || addr[:2] == "0X" {
		add, err := address.FromHex(addr)
		if err != nil {
			return nil, err
		}

		addr = add.String()
	}
	offset := req.GetOffset()
	size := req.GetSize()
	if size == 0 {
		size = 25
	}
	sort := req.GetSort()
	if sort != "asc" && sort != "desc" {
		sort = "asc"
	}

	var count int64
	err := db.Table("block_action a").Where("a.from=? or a.to=?", addr, addr).Count(&count).Error

	if err != nil {
		return nil, err
	}
	resp.Count = uint64(count)

	query := "SELECT a.action_hash,a.action_type,a.block_height,a.from,a.to,a.gas_price*r.gas_consumed,a.gas_limit,a.nonce,a.amount,r.status,b.block_hash,b.timestamp FROM block_action a inner join block b on b.block_height=a.block_height inner join block_receipt r on r.action_hash=a.action_hash where a.from=? or a.to=? order by a.id " + sort + " limit ? offset ?"
	rows, err := db.Raw(query, addr, addr, size, offset).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var actHash, actType, from, to, blkHash, gasFee, amount string
	var blkHeight, gasLimit, nonce, status, timestamp uint64
	for rows.Next() {
		if err := rows.Scan(&actHash, &actType, &blkHeight, &from, &to, &gasFee, &gasLimit, &nonce, &amount, &status, &blkHash, &timestamp); err != nil {
			return nil, err
		}
		resp.Results = append(resp.Results, &api.ActionsByAddressResult{
			ActHash:   actHash,
			ActType:   actType,
			TimeStamp: timestamp,
			BlkHash:   blkHash,
			Sender:    from,
			Recipient: to,
			Amount:    amount,
			GasFee:    gasFee,
		})
	}
	return resp, nil
}

func (s *ActionsService) GetXrc20ByAddress(ctx context.Context, req *api.ActionsRequest) (*api.Xrc20ByAddressResponse, error) {
	return s.getXrcByAddress(Xrc20, req)
}

func (s *ActionsService) GetXrc721ByAddress(ctx context.Context, req *api.ActionsRequest) (*api.Xrc20ByAddressResponse, error) {
	return s.getXrcByAddress(Xrc721, req)
}

func (s *ActionsService) getXrcByAddress(xrcType XrcType, req *api.ActionsRequest) (*api.Xrc20ByAddressResponse, error) {
	var xrcTable string
	switch xrcType {
	case Xrc20:
		xrcTable = "token_erc20 a"
	case Xrc721:
		xrcTable = "token_erc721 a"
	}
	resp := &api.Xrc20ByAddressResponse{
		Count: 0,
	}
	db := db.DB()
	addr := req.GetAddress()
	if addr[:2] == "0x" || addr[:2] == "0X" {
		add, err := address.FromHex(addr)
		if err != nil {
			return nil, err
		}

		addr = add.String()
	}
	offset := req.GetOffset()
	size := req.GetSize()
	if size == 0 {
		size = 25
	}
	sort := req.GetSort()
	if sort != "asc" && sort != "desc" {
		sort = "asc"
	}

	var count int64
	err := db.Table(xrcTable).Where("a.from=? or a.to=?", addr, addr).Count(&count).Error

	if err != nil {
		return nil, err
	}
	resp.Count = uint64(count)

	query := "select t.*,(select timestamp from block where block_height=t.block_height) from (select a.block_height,a.action_hash,a.contract_address,a.amount,a.from,a.to from " + xrcTable + " where a.from=? or a.to=? order by a.id " + sort + " limit ? offset ?) t"
	rows, err := db.Raw(query, addr, addr, size, offset).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var actHash, contractAddress, from, to, amount string
	var blkHeight, timestamp uint64
	for rows.Next() {
		if err := rows.Scan(&blkHeight, &actHash, &contractAddress, &amount, &from, &to, &timestamp); err != nil {
			return nil, err
		}
		resp.Results = append(resp.Results, &api.Xrc20ByAddressResult{
			ActHash:         actHash,
			BlkHeight:       blkHeight,
			TimeStamp:       timestamp,
			ContractAddress: contractAddress,
			From:            from,
			To:              to,
			Amount:          amount,
		})
	}
	return resp, nil
}
