package apiservice

import (
	"context"

	"github.com/iotexproject/iotex-address/address"
	"github.com/iotexproject/iotex-analyser/api"
	"github.com/iotexproject/iotex-analyser/db"
)

type ActionsService struct {
	api.UnimplementedActionsServiceServer
}

// type rawActionByAddr struct {
// 	ActionHash  string
// 	ActionType  string
// 	BlockHeight uint64
// 	From        string
// 	To          string
// 	GasPrice    string
// 	GasLimit    uint64
// 	GasConsumed uint64
// 	Nonce       uint32
// 	Amount      string
// 	Status      int32
// 	BlockHash   string
// 	Timestamp   uint64
// }

//curl -d '{"address": "io102s4660k3cynae2r8gde6scg74mf6f7k9dq955", "height":11900487 }' http://127.0.0.1:7778/api.ActionsService.GetActionsByAddress
func (s *ActionsService) GetActionsByAddress(ctx context.Context, req *api.ActionsRequest) (*api.ActionsByAddressResponse, error) {
	resp := &api.ActionsByAddressResponse{
		Total: 0,
	}
	db := db.DB().Debug()
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

	var count int64
	err := db.Table("block_action a").Where("a.from=? or a.to=?", addr, addr).Count(&count).Error

	if err != nil {
		return nil, err
	}
	resp.Total = uint64(count)
	query := "SELECT a.action_hash,a.action_type,a.block_height,a.from,a.to,a.gas_price*r.gas_consumed,a.gas_limit,a.nonce,a.amount,r.status,b.block_hash,b.timestamp FROM block_action a, (select id from (SELECT a.id FROM block_action a WHERE a.from=? union SELECT a.id FROM block_action a WHERE a.to=?) tmp order by id desc limit ? offset ?) aa, block b, block_receipt r where a.id=aa.id and b.block_height=a.block_height and r.action_hash=a.action_hash"
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
