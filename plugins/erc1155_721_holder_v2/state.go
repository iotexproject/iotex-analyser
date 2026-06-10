package main

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const holderChunkSize = 500

type holderOperationType int

const (
	holderOpERC721Mint holderOperationType = iota
	holderOpERC721Burn
	holderOpERC721Transfer
	holderOpERC1155Mint
	holderOpERC1155Burn
	holderOpERC1155Transfer
)

type holderOperation struct {
	opType   holderOperationType
	contract string
	from     string
	to       string
	tokenID  string
	ercType  uint16
	value    decimal.Decimal
}

type holderKey struct {
	contract string
	holder   string
	tokenID  string
}

type holderTokenKey struct {
	contract string
	tokenID  string
}

type holderState struct {
	ercType uint16
	value   decimal.Decimal
}

func applyHolderOperations(tx *gorm.DB, ops []holderOperation) error {
	if len(ops) == 0 {
		return nil
	}

	currentStates, err := loadExistingHolderStates(tx, collectTouchedKeys(ops))
	if err != nil {
		return err
	}

	deleteByKey := make(map[holderKey]struct{})
	deleteByToken := make(map[holderTokenKey]struct{})

	for _, op := range ops {
		tokenKey := holderTokenKey{contract: op.contract, tokenID: op.tokenID}
		switch op.opType {
		case holderOpERC721Mint:
			delete(deleteByToken, tokenKey)
			currentStates[holderKey{contract: op.contract, holder: op.to, tokenID: op.tokenID}] = holderState{
				ercType: op.ercType,
				value:   decimal.NewFromInt(1),
			}
		case holderOpERC721Burn:
			deleteByToken[tokenKey] = struct{}{}
			for key := range currentStates {
				if key.contract == op.contract && key.tokenID == op.tokenID {
					delete(currentStates, key)
				}
			}
		case holderOpERC721Transfer:
			delete(deleteByToken, tokenKey)
			fromKey := holderKey{contract: op.contract, holder: op.from, tokenID: op.tokenID}
			deleteByKey[fromKey] = struct{}{}
			delete(currentStates, fromKey)
			currentStates[holderKey{contract: op.contract, holder: op.to, tokenID: op.tokenID}] = holderState{
				ercType: op.ercType,
				value:   decimal.NewFromInt(1),
			}
		case holderOpERC1155Mint:
			applyPositiveDelta(currentStates, holderKey{contract: op.contract, holder: op.to, tokenID: op.tokenID}, op.ercType, op.value)
		case holderOpERC1155Burn:
			if err := applyNegativeDelta(currentStates, deleteByKey, holderKey{contract: op.contract, holder: op.from, tokenID: op.tokenID}, op.value); err != nil {
				return err
			}
		case holderOpERC1155Transfer:
			if err := applyNegativeDelta(currentStates, deleteByKey, holderKey{contract: op.contract, holder: op.from, tokenID: op.tokenID}, op.value); err != nil {
				return err
			}
			applyPositiveDelta(currentStates, holderKey{contract: op.contract, holder: op.to, tokenID: op.tokenID}, op.ercType, op.value)
		}
	}

	if err := deleteHoldersByToken(tx, deleteByToken); err != nil {
		return err
	}
	if err := deleteHoldersByKey(tx, deleteByKey); err != nil {
		return err
	}
	return upsertHolderStates(tx, currentStates)
}

func applyPositiveDelta(states map[holderKey]holderState, key holderKey, ercType uint16, value decimal.Decimal) {
	state := states[key]
	state.ercType = ercType
	state.value = state.value.Add(value)
	states[key] = state
}

func applyNegativeDelta(states map[holderKey]holderState, deleteByKey map[holderKey]struct{}, key holderKey, value decimal.Decimal) error {
	state, ok := states[key]
	if !ok {
		// return errors.Errorf("holder state not found for contract=%s holder=%s tokenID=%s", key.contract, key.holder, key.tokenID)
		return nil
	}
	if state.value.LessThan(value) {
		return errors.Errorf(
			"holder balance underflow for contract=%s holder=%s tokenID=%s: have=%s need=%s",
			key.contract,
			key.holder,
			key.tokenID,
			state.value.String(),
			value.String(),
		)
	}
	if state.value.Equal(value) {
		delete(states, key)
		deleteByKey[key] = struct{}{}
		return nil
	}
	state.value = state.value.Sub(value)
	states[key] = state
	return nil
}

func collectTouchedKeys(ops []holderOperation) []holderKey {
	keys := make([]holderKey, 0)
	seen := make(map[holderKey]struct{})
	for _, op := range ops {
		switch op.opType {
		case holderOpERC1155Mint:
			key := holderKey{contract: op.contract, holder: op.to, tokenID: op.tokenID}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			keys = append(keys, key)
		case holderOpERC1155Burn:
			key := holderKey{contract: op.contract, holder: op.from, tokenID: op.tokenID}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			keys = append(keys, key)
		case holderOpERC1155Transfer:
			fromKey := holderKey{contract: op.contract, holder: op.from, tokenID: op.tokenID}
			if _, ok := seen[fromKey]; !ok {
				seen[fromKey] = struct{}{}
				keys = append(keys, fromKey)
			}
			toKey := holderKey{contract: op.contract, holder: op.to, tokenID: op.tokenID}
			if _, ok := seen[toKey]; ok {
				continue
			}
			seen[toKey] = struct{}{}
			keys = append(keys, toKey)
		}
	}
	return keys
}

func loadExistingHolderStates(tx *gorm.DB, keys []holderKey) (map[holderKey]holderState, error) {
	states := make(map[holderKey]holderState, len(keys))
	if len(keys) == 0 {
		return states, nil
	}
	tableName := Erc1155721Holder{}.TableName()
	for start := 0; start < len(keys); start += holderChunkSize {
		end := start + holderChunkSize
		if end > len(keys) {
			end = len(keys)
		}
		batch := keys[start:end]
		query, args := buildHolderTupleQuery(
			fmt.Sprintf("SELECT contract_address, holder, erc_type, token_id, token_value FROM %s WHERE (contract_address, holder, token_id) IN ", tableName),
			batch,
		)
		var rows []Erc1155721Holder
		if err := tx.Raw(query, args...).Scan(&rows).Error; err != nil {
			return nil, err
		}
		for _, row := range rows {
			states[holderKey{
				contract: row.ContractAddress,
				holder:   row.Holder,
				tokenID:  row.TokenID.String(),
			}] = holderState{
				ercType: row.ErcType,
				value:   row.TokenValue,
			}
		}
	}
	return states, nil
}

func deleteHoldersByKey(tx *gorm.DB, keys map[holderKey]struct{}) error {
	if len(keys) == 0 {
		return nil
	}
	keyList := make([]holderKey, 0, len(keys))
	for key := range keys {
		keyList = append(keyList, key)
	}
	tableName := Erc1155721Holder{}.TableName()
	for start := 0; start < len(keyList); start += holderChunkSize {
		end := start + holderChunkSize
		if end > len(keyList) {
			end = len(keyList)
		}
		query, args := buildHolderTupleQuery(
			fmt.Sprintf("DELETE FROM %s WHERE (contract_address, holder, token_id) IN ", tableName),
			keyList[start:end],
		)
		if err := tx.Exec(query, args...).Error; err != nil {
			return err
		}
	}
	return nil
}

func deleteHoldersByToken(tx *gorm.DB, tokens map[holderTokenKey]struct{}) error {
	if len(tokens) == 0 {
		return nil
	}
	tokenList := make([]holderTokenKey, 0, len(tokens))
	for token := range tokens {
		tokenList = append(tokenList, token)
	}
	tableName := Erc1155721Holder{}.TableName()
	for start := 0; start < len(tokenList); start += holderChunkSize {
		end := start + holderChunkSize
		if end > len(tokenList) {
			end = len(tokenList)
		}
		query, args := buildHolderTokenTupleQuery(
			fmt.Sprintf("DELETE FROM %s WHERE (contract_address, token_id) IN ", tableName),
			tokenList[start:end],
		)
		if err := tx.Exec(query, args...).Error; err != nil {
			return err
		}
	}
	return nil
}

func upsertHolderStates(tx *gorm.DB, states map[holderKey]holderState) error {
	if len(states) == 0 {
		return nil
	}
	rows := make([]Erc1155721Holder, 0, len(states))
	for key, state := range states {
		rows = append(rows, Erc1155721Holder{
			ContractAddress: key.contract,
			Holder:          key.holder,
			ErcType:         state.ercType,
			TokenID:         decimal.RequireFromString(key.tokenID),
			TokenValue:      state.value,
		})
	}
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "contract_address"}, {Name: "holder"}, {Name: "token_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"erc_type", "token_value"}),
	}).CreateInBatches(rows, holderChunkSize).Error
}

func buildHolderTupleQuery(prefix string, keys []holderKey) (string, []interface{}) {
	var builder strings.Builder
	args := make([]interface{}, 0, len(keys)*3)
	builder.WriteString(prefix)
	builder.WriteString("(")
	for i, key := range keys {
		if i > 0 {
			builder.WriteString(",")
		}
		builder.WriteString("(?,?,?)")
		args = append(args, key.contract, key.holder, decimal.RequireFromString(key.tokenID))
	}
	builder.WriteString(")")
	return builder.String(), args
}

func buildHolderTokenTupleQuery(prefix string, tokens []holderTokenKey) (string, []interface{}) {
	var builder strings.Builder
	args := make([]interface{}, 0, len(tokens)*2)
	builder.WriteString(prefix)
	builder.WriteString("(")
	for i, token := range tokens {
		if i > 0 {
			builder.WriteString(",")
		}
		builder.WriteString("(?,?)")
		args = append(args, token.contract, decimal.RequireFromString(token.tokenID))
	}
	builder.WriteString(")")
	return builder.String(), args
}
