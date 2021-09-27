// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package main

import (
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// Struct0 is an auto generated low-level Go binding around an user-defined struct.
type Struct0 struct {
	Konstante     *big.Int
	StartBlock    *big.Int
	EndBlock      *big.Int
	LastDripBlock *big.Int
	Volume        *big.Int
	CloudVolume   *big.Int
	Exists        bool
}

// ExchangeABI is the input ABI used to generate the binding from.
const ExchangeABI = "[{\"constant\":true,\"inputs\":[{\"name\":\"_asset\",\"type\":\"address\"}],\"name\":\"volumeToDripPerBlock\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"safe\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"volumeScale\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_asset\",\"type\":\"address\"},{\"name\":\"_minAmount\",\"type\":\"uint256\"},{\"name\":\"_points\",\"type\":\"uint256\"}],\"name\":\"redeem\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"accountant\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_asset\",\"type\":\"address\"},{\"name\":\"_endBlock\",\"type\":\"uint256\"}],\"name\":\"extendDuration\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_asset\",\"type\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"redeemExactAssetCost\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_asset\",\"type\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\"},{\"name\":\"_maxPoints\",\"type\":\"uint256\"}],\"name\":\"redeemExactAsset\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_asset\",\"type\":\"address\"}],\"name\":\"drip\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"maxDuration\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_asset\",\"type\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\"},{\"name\":\"_duration\",\"type\":\"uint256\"},{\"name\":\"_konstante\",\"type\":\"uint256\"}],\"name\":\"addAsset\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_asset\",\"type\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"increaseSupply\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"isOwner\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_asset\",\"type\":\"address\"}],\"name\":\"poolOf\",\"outputs\":[{\"components\":[{\"name\":\"konstante\",\"type\":\"uint256\"},{\"name\":\"startBlock\",\"type\":\"uint256\"},{\"name\":\"endBlock\",\"type\":\"uint256\"},{\"name\":\"lastDripBlock\",\"type\":\"uint256\"},{\"name\":\"volume\",\"type\":\"uint256\"},{\"name\":\"cloudVolume\",\"type\":\"uint256\"},{\"name\":\"exists\",\"type\":\"bool\"}],\"name\":\"\",\"type\":\"tuple\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"strategy\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_asset\",\"type\":\"address\"},{\"name\":\"_points\",\"type\":\"uint256\"}],\"name\":\"redeemAmount\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"assets\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_newStrategy\",\"type\":\"address\"}],\"name\":\"updateStrategy\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"numOfAssets\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_accountant\",\"type\":\"address\"},{\"name\":\"_safe\",\"type\":\"address\"},{\"name\":\"_strategy\",\"type\":\"address\"},{\"name\":\"_maxDuration\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"user\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"asset\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"points\",\"type\":\"uint256\"}],\"name\":\"Redemption\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"provider\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"asset\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"endBlock\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"konstante\",\"type\":\"uint256\"}],\"name\":\"AssetAdded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"asset\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"volume\",\"type\":\"uint256\"}],\"name\":\"AssetDripped\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"strategy\",\"type\":\"address\"}],\"name\":\"StrategyUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"}]"

// Exchange is an auto generated Go binding around an Ethereum contract.
type Exchange struct {
	ExchangeCaller     // Read-only binding to the contract
	ExchangeTransactor // Write-only binding to the contract
	ExchangeFilterer   // Log filterer for contract events
}

// ExchangeCaller is an auto generated read-only Go binding around an Ethereum contract.
type ExchangeCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ExchangeTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ExchangeTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ExchangeFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ExchangeFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ExchangeSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ExchangeSession struct {
	Contract     *Exchange         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ExchangeCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ExchangeCallerSession struct {
	Contract *ExchangeCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// ExchangeTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ExchangeTransactorSession struct {
	Contract     *ExchangeTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// ExchangeRaw is an auto generated low-level Go binding around an Ethereum contract.
type ExchangeRaw struct {
	Contract *Exchange // Generic contract binding to access the raw methods on
}

// ExchangeCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ExchangeCallerRaw struct {
	Contract *ExchangeCaller // Generic read-only contract binding to access the raw methods on
}

// ExchangeTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ExchangeTransactorRaw struct {
	Contract *ExchangeTransactor // Generic write-only contract binding to access the raw methods on
}

// NewExchange creates a new instance of Exchange, bound to a specific deployed contract.
func NewExchange(address common.Address, backend bind.ContractBackend) (*Exchange, error) {
	contract, err := bindExchange(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Exchange{ExchangeCaller: ExchangeCaller{contract: contract}, ExchangeTransactor: ExchangeTransactor{contract: contract}, ExchangeFilterer: ExchangeFilterer{contract: contract}}, nil
}

// NewExchangeCaller creates a new read-only instance of Exchange, bound to a specific deployed contract.
func NewExchangeCaller(address common.Address, caller bind.ContractCaller) (*ExchangeCaller, error) {
	contract, err := bindExchange(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ExchangeCaller{contract: contract}, nil
}

// NewExchangeTransactor creates a new write-only instance of Exchange, bound to a specific deployed contract.
func NewExchangeTransactor(address common.Address, transactor bind.ContractTransactor) (*ExchangeTransactor, error) {
	contract, err := bindExchange(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ExchangeTransactor{contract: contract}, nil
}

// NewExchangeFilterer creates a new log filterer instance of Exchange, bound to a specific deployed contract.
func NewExchangeFilterer(address common.Address, filterer bind.ContractFilterer) (*ExchangeFilterer, error) {
	contract, err := bindExchange(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ExchangeFilterer{contract: contract}, nil
}

// bindExchange binds a generic wrapper to an already deployed contract.
func bindExchange(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(ExchangeABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Exchange *ExchangeRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Exchange.Contract.ExchangeCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Exchange *ExchangeRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Exchange.Contract.ExchangeTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Exchange *ExchangeRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Exchange.Contract.ExchangeTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Exchange *ExchangeCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Exchange.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Exchange *ExchangeTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Exchange.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Exchange *ExchangeTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Exchange.Contract.contract.Transact(opts, method, params...)
}

// Accountant is a free data retrieval call binding the contract method 0x4fb3ccc5.
//
// Solidity: function accountant() view returns(address)
func (_Exchange *ExchangeCaller) Accountant(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Exchange.contract.Call(opts, &out, "accountant")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Accountant is a free data retrieval call binding the contract method 0x4fb3ccc5.
//
// Solidity: function accountant() view returns(address)
func (_Exchange *ExchangeSession) Accountant() (common.Address, error) {
	return _Exchange.Contract.Accountant(&_Exchange.CallOpts)
}

// Accountant is a free data retrieval call binding the contract method 0x4fb3ccc5.
//
// Solidity: function accountant() view returns(address)
func (_Exchange *ExchangeCallerSession) Accountant() (common.Address, error) {
	return _Exchange.Contract.Accountant(&_Exchange.CallOpts)
}

// Assets is a free data retrieval call binding the contract method 0xcf35bdd0.
//
// Solidity: function assets(uint256 ) view returns(address)
func (_Exchange *ExchangeCaller) Assets(opts *bind.CallOpts, arg0 *big.Int) (common.Address, error) {
	var out []interface{}
	err := _Exchange.contract.Call(opts, &out, "assets", arg0)

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Assets is a free data retrieval call binding the contract method 0xcf35bdd0.
//
// Solidity: function assets(uint256 ) view returns(address)
func (_Exchange *ExchangeSession) Assets(arg0 *big.Int) (common.Address, error) {
	return _Exchange.Contract.Assets(&_Exchange.CallOpts, arg0)
}

// Assets is a free data retrieval call binding the contract method 0xcf35bdd0.
//
// Solidity: function assets(uint256 ) view returns(address)
func (_Exchange *ExchangeCallerSession) Assets(arg0 *big.Int) (common.Address, error) {
	return _Exchange.Contract.Assets(&_Exchange.CallOpts, arg0)
}

// IsOwner is a free data retrieval call binding the contract method 0x8f32d59b.
//
// Solidity: function isOwner() view returns(bool)
func (_Exchange *ExchangeCaller) IsOwner(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _Exchange.contract.Call(opts, &out, "isOwner")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsOwner is a free data retrieval call binding the contract method 0x8f32d59b.
//
// Solidity: function isOwner() view returns(bool)
func (_Exchange *ExchangeSession) IsOwner() (bool, error) {
	return _Exchange.Contract.IsOwner(&_Exchange.CallOpts)
}

// IsOwner is a free data retrieval call binding the contract method 0x8f32d59b.
//
// Solidity: function isOwner() view returns(bool)
func (_Exchange *ExchangeCallerSession) IsOwner() (bool, error) {
	return _Exchange.Contract.IsOwner(&_Exchange.CallOpts)
}

// MaxDuration is a free data retrieval call binding the contract method 0x6db5c8fd.
//
// Solidity: function maxDuration() view returns(uint256)
func (_Exchange *ExchangeCaller) MaxDuration(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Exchange.contract.Call(opts, &out, "maxDuration")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MaxDuration is a free data retrieval call binding the contract method 0x6db5c8fd.
//
// Solidity: function maxDuration() view returns(uint256)
func (_Exchange *ExchangeSession) MaxDuration() (*big.Int, error) {
	return _Exchange.Contract.MaxDuration(&_Exchange.CallOpts)
}

// MaxDuration is a free data retrieval call binding the contract method 0x6db5c8fd.
//
// Solidity: function maxDuration() view returns(uint256)
func (_Exchange *ExchangeCallerSession) MaxDuration() (*big.Int, error) {
	return _Exchange.Contract.MaxDuration(&_Exchange.CallOpts)
}

// NumOfAssets is a free data retrieval call binding the contract method 0xf2cf057f.
//
// Solidity: function numOfAssets() view returns(uint256)
func (_Exchange *ExchangeCaller) NumOfAssets(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Exchange.contract.Call(opts, &out, "numOfAssets")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NumOfAssets is a free data retrieval call binding the contract method 0xf2cf057f.
//
// Solidity: function numOfAssets() view returns(uint256)
func (_Exchange *ExchangeSession) NumOfAssets() (*big.Int, error) {
	return _Exchange.Contract.NumOfAssets(&_Exchange.CallOpts)
}

// NumOfAssets is a free data retrieval call binding the contract method 0xf2cf057f.
//
// Solidity: function numOfAssets() view returns(uint256)
func (_Exchange *ExchangeCallerSession) NumOfAssets() (*big.Int, error) {
	return _Exchange.Contract.NumOfAssets(&_Exchange.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Exchange *ExchangeCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Exchange.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Exchange *ExchangeSession) Owner() (common.Address, error) {
	return _Exchange.Contract.Owner(&_Exchange.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Exchange *ExchangeCallerSession) Owner() (common.Address, error) {
	return _Exchange.Contract.Owner(&_Exchange.CallOpts)
}

// PoolOf is a free data retrieval call binding the contract method 0x988b1fa7.
//
// Solidity: function poolOf(address _asset) view returns((uint256,uint256,uint256,uint256,uint256,uint256,bool))
func (_Exchange *ExchangeCaller) PoolOf(opts *bind.CallOpts, _asset common.Address) (Struct0, error) {
	var out []interface{}
	err := _Exchange.contract.Call(opts, &out, "poolOf", _asset)

	if err != nil {
		return *new(Struct0), err
	}

	out0 := *abi.ConvertType(out[0], new(Struct0)).(*Struct0)

	return out0, err

}

// PoolOf is a free data retrieval call binding the contract method 0x988b1fa7.
//
// Solidity: function poolOf(address _asset) view returns((uint256,uint256,uint256,uint256,uint256,uint256,bool))
func (_Exchange *ExchangeSession) PoolOf(_asset common.Address) (Struct0, error) {
	return _Exchange.Contract.PoolOf(&_Exchange.CallOpts, _asset)
}

// PoolOf is a free data retrieval call binding the contract method 0x988b1fa7.
//
// Solidity: function poolOf(address _asset) view returns((uint256,uint256,uint256,uint256,uint256,uint256,bool))
func (_Exchange *ExchangeCallerSession) PoolOf(_asset common.Address) (Struct0, error) {
	return _Exchange.Contract.PoolOf(&_Exchange.CallOpts, _asset)
}

// RedeemAmount is a free data retrieval call binding the contract method 0xb82ac6ae.
//
// Solidity: function redeemAmount(address _asset, uint256 _points) view returns(uint256)
func (_Exchange *ExchangeCaller) RedeemAmount(opts *bind.CallOpts, _asset common.Address, _points *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Exchange.contract.Call(opts, &out, "redeemAmount", _asset, _points)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// RedeemAmount is a free data retrieval call binding the contract method 0xb82ac6ae.
//
// Solidity: function redeemAmount(address _asset, uint256 _points) view returns(uint256)
func (_Exchange *ExchangeSession) RedeemAmount(_asset common.Address, _points *big.Int) (*big.Int, error) {
	return _Exchange.Contract.RedeemAmount(&_Exchange.CallOpts, _asset, _points)
}

// RedeemAmount is a free data retrieval call binding the contract method 0xb82ac6ae.
//
// Solidity: function redeemAmount(address _asset, uint256 _points) view returns(uint256)
func (_Exchange *ExchangeCallerSession) RedeemAmount(_asset common.Address, _points *big.Int) (*big.Int, error) {
	return _Exchange.Contract.RedeemAmount(&_Exchange.CallOpts, _asset, _points)
}

// RedeemExactAssetCost is a free data retrieval call binding the contract method 0x584884c7.
//
// Solidity: function redeemExactAssetCost(address _asset, uint256 _amount) view returns(uint256)
func (_Exchange *ExchangeCaller) RedeemExactAssetCost(opts *bind.CallOpts, _asset common.Address, _amount *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _Exchange.contract.Call(opts, &out, "redeemExactAssetCost", _asset, _amount)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// RedeemExactAssetCost is a free data retrieval call binding the contract method 0x584884c7.
//
// Solidity: function redeemExactAssetCost(address _asset, uint256 _amount) view returns(uint256)
func (_Exchange *ExchangeSession) RedeemExactAssetCost(_asset common.Address, _amount *big.Int) (*big.Int, error) {
	return _Exchange.Contract.RedeemExactAssetCost(&_Exchange.CallOpts, _asset, _amount)
}

// RedeemExactAssetCost is a free data retrieval call binding the contract method 0x584884c7.
//
// Solidity: function redeemExactAssetCost(address _asset, uint256 _amount) view returns(uint256)
func (_Exchange *ExchangeCallerSession) RedeemExactAssetCost(_asset common.Address, _amount *big.Int) (*big.Int, error) {
	return _Exchange.Contract.RedeemExactAssetCost(&_Exchange.CallOpts, _asset, _amount)
}

// Safe is a free data retrieval call binding the contract method 0x186f0354.
//
// Solidity: function safe() view returns(address)
func (_Exchange *ExchangeCaller) Safe(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Exchange.contract.Call(opts, &out, "safe")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Safe is a free data retrieval call binding the contract method 0x186f0354.
//
// Solidity: function safe() view returns(address)
func (_Exchange *ExchangeSession) Safe() (common.Address, error) {
	return _Exchange.Contract.Safe(&_Exchange.CallOpts)
}

// Safe is a free data retrieval call binding the contract method 0x186f0354.
//
// Solidity: function safe() view returns(address)
func (_Exchange *ExchangeCallerSession) Safe() (common.Address, error) {
	return _Exchange.Contract.Safe(&_Exchange.CallOpts)
}

// Strategy is a free data retrieval call binding the contract method 0xa8c62e76.
//
// Solidity: function strategy() view returns(address)
func (_Exchange *ExchangeCaller) Strategy(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Exchange.contract.Call(opts, &out, "strategy")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Strategy is a free data retrieval call binding the contract method 0xa8c62e76.
//
// Solidity: function strategy() view returns(address)
func (_Exchange *ExchangeSession) Strategy() (common.Address, error) {
	return _Exchange.Contract.Strategy(&_Exchange.CallOpts)
}

// Strategy is a free data retrieval call binding the contract method 0xa8c62e76.
//
// Solidity: function strategy() view returns(address)
func (_Exchange *ExchangeCallerSession) Strategy() (common.Address, error) {
	return _Exchange.Contract.Strategy(&_Exchange.CallOpts)
}

// VolumeScale is a free data retrieval call binding the contract method 0x2a541bdb.
//
// Solidity: function volumeScale() view returns(uint256)
func (_Exchange *ExchangeCaller) VolumeScale(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Exchange.contract.Call(opts, &out, "volumeScale")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// VolumeScale is a free data retrieval call binding the contract method 0x2a541bdb.
//
// Solidity: function volumeScale() view returns(uint256)
func (_Exchange *ExchangeSession) VolumeScale() (*big.Int, error) {
	return _Exchange.Contract.VolumeScale(&_Exchange.CallOpts)
}

// VolumeScale is a free data retrieval call binding the contract method 0x2a541bdb.
//
// Solidity: function volumeScale() view returns(uint256)
func (_Exchange *ExchangeCallerSession) VolumeScale() (*big.Int, error) {
	return _Exchange.Contract.VolumeScale(&_Exchange.CallOpts)
}

// VolumeToDripPerBlock is a free data retrieval call binding the contract method 0x166ebd04.
//
// Solidity: function volumeToDripPerBlock(address _asset) view returns(uint256)
func (_Exchange *ExchangeCaller) VolumeToDripPerBlock(opts *bind.CallOpts, _asset common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Exchange.contract.Call(opts, &out, "volumeToDripPerBlock", _asset)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// VolumeToDripPerBlock is a free data retrieval call binding the contract method 0x166ebd04.
//
// Solidity: function volumeToDripPerBlock(address _asset) view returns(uint256)
func (_Exchange *ExchangeSession) VolumeToDripPerBlock(_asset common.Address) (*big.Int, error) {
	return _Exchange.Contract.VolumeToDripPerBlock(&_Exchange.CallOpts, _asset)
}

// VolumeToDripPerBlock is a free data retrieval call binding the contract method 0x166ebd04.
//
// Solidity: function volumeToDripPerBlock(address _asset) view returns(uint256)
func (_Exchange *ExchangeCallerSession) VolumeToDripPerBlock(_asset common.Address) (*big.Int, error) {
	return _Exchange.Contract.VolumeToDripPerBlock(&_Exchange.CallOpts, _asset)
}

// AddAsset is a paid mutator transaction binding the contract method 0x75a1608d.
//
// Solidity: function addAsset(address _asset, uint256 _amount, uint256 _duration, uint256 _konstante) returns()
func (_Exchange *ExchangeTransactor) AddAsset(opts *bind.TransactOpts, _asset common.Address, _amount *big.Int, _duration *big.Int, _konstante *big.Int) (*types.Transaction, error) {
	return _Exchange.contract.Transact(opts, "addAsset", _asset, _amount, _duration, _konstante)
}

// AddAsset is a paid mutator transaction binding the contract method 0x75a1608d.
//
// Solidity: function addAsset(address _asset, uint256 _amount, uint256 _duration, uint256 _konstante) returns()
func (_Exchange *ExchangeSession) AddAsset(_asset common.Address, _amount *big.Int, _duration *big.Int, _konstante *big.Int) (*types.Transaction, error) {
	return _Exchange.Contract.AddAsset(&_Exchange.TransactOpts, _asset, _amount, _duration, _konstante)
}

// AddAsset is a paid mutator transaction binding the contract method 0x75a1608d.
//
// Solidity: function addAsset(address _asset, uint256 _amount, uint256 _duration, uint256 _konstante) returns()
func (_Exchange *ExchangeTransactorSession) AddAsset(_asset common.Address, _amount *big.Int, _duration *big.Int, _konstante *big.Int) (*types.Transaction, error) {
	return _Exchange.Contract.AddAsset(&_Exchange.TransactOpts, _asset, _amount, _duration, _konstante)
}

// Drip is a paid mutator transaction binding the contract method 0x67a5cd06.
//
// Solidity: function drip(address _asset) returns()
func (_Exchange *ExchangeTransactor) Drip(opts *bind.TransactOpts, _asset common.Address) (*types.Transaction, error) {
	return _Exchange.contract.Transact(opts, "drip", _asset)
}

// Drip is a paid mutator transaction binding the contract method 0x67a5cd06.
//
// Solidity: function drip(address _asset) returns()
func (_Exchange *ExchangeSession) Drip(_asset common.Address) (*types.Transaction, error) {
	return _Exchange.Contract.Drip(&_Exchange.TransactOpts, _asset)
}

// Drip is a paid mutator transaction binding the contract method 0x67a5cd06.
//
// Solidity: function drip(address _asset) returns()
func (_Exchange *ExchangeTransactorSession) Drip(_asset common.Address) (*types.Transaction, error) {
	return _Exchange.Contract.Drip(&_Exchange.TransactOpts, _asset)
}

// ExtendDuration is a paid mutator transaction binding the contract method 0x5146bb5d.
//
// Solidity: function extendDuration(address _asset, uint256 _endBlock) returns()
func (_Exchange *ExchangeTransactor) ExtendDuration(opts *bind.TransactOpts, _asset common.Address, _endBlock *big.Int) (*types.Transaction, error) {
	return _Exchange.contract.Transact(opts, "extendDuration", _asset, _endBlock)
}

// ExtendDuration is a paid mutator transaction binding the contract method 0x5146bb5d.
//
// Solidity: function extendDuration(address _asset, uint256 _endBlock) returns()
func (_Exchange *ExchangeSession) ExtendDuration(_asset common.Address, _endBlock *big.Int) (*types.Transaction, error) {
	return _Exchange.Contract.ExtendDuration(&_Exchange.TransactOpts, _asset, _endBlock)
}

// ExtendDuration is a paid mutator transaction binding the contract method 0x5146bb5d.
//
// Solidity: function extendDuration(address _asset, uint256 _endBlock) returns()
func (_Exchange *ExchangeTransactorSession) ExtendDuration(_asset common.Address, _endBlock *big.Int) (*types.Transaction, error) {
	return _Exchange.Contract.ExtendDuration(&_Exchange.TransactOpts, _asset, _endBlock)
}

// IncreaseSupply is a paid mutator transaction binding the contract method 0x79fcd8ee.
//
// Solidity: function increaseSupply(address _asset, uint256 _amount) returns()
func (_Exchange *ExchangeTransactor) IncreaseSupply(opts *bind.TransactOpts, _asset common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _Exchange.contract.Transact(opts, "increaseSupply", _asset, _amount)
}

// IncreaseSupply is a paid mutator transaction binding the contract method 0x79fcd8ee.
//
// Solidity: function increaseSupply(address _asset, uint256 _amount) returns()
func (_Exchange *ExchangeSession) IncreaseSupply(_asset common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _Exchange.Contract.IncreaseSupply(&_Exchange.TransactOpts, _asset, _amount)
}

// IncreaseSupply is a paid mutator transaction binding the contract method 0x79fcd8ee.
//
// Solidity: function increaseSupply(address _asset, uint256 _amount) returns()
func (_Exchange *ExchangeTransactorSession) IncreaseSupply(_asset common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _Exchange.Contract.IncreaseSupply(&_Exchange.TransactOpts, _asset, _amount)
}

// Redeem is a paid mutator transaction binding the contract method 0x2b83cccd.
//
// Solidity: function redeem(address _asset, uint256 _minAmount, uint256 _points) returns()
func (_Exchange *ExchangeTransactor) Redeem(opts *bind.TransactOpts, _asset common.Address, _minAmount *big.Int, _points *big.Int) (*types.Transaction, error) {
	return _Exchange.contract.Transact(opts, "redeem", _asset, _minAmount, _points)
}

// Redeem is a paid mutator transaction binding the contract method 0x2b83cccd.
//
// Solidity: function redeem(address _asset, uint256 _minAmount, uint256 _points) returns()
func (_Exchange *ExchangeSession) Redeem(_asset common.Address, _minAmount *big.Int, _points *big.Int) (*types.Transaction, error) {
	return _Exchange.Contract.Redeem(&_Exchange.TransactOpts, _asset, _minAmount, _points)
}

// Redeem is a paid mutator transaction binding the contract method 0x2b83cccd.
//
// Solidity: function redeem(address _asset, uint256 _minAmount, uint256 _points) returns()
func (_Exchange *ExchangeTransactorSession) Redeem(_asset common.Address, _minAmount *big.Int, _points *big.Int) (*types.Transaction, error) {
	return _Exchange.Contract.Redeem(&_Exchange.TransactOpts, _asset, _minAmount, _points)
}

// RedeemExactAsset is a paid mutator transaction binding the contract method 0x5b302d09.
//
// Solidity: function redeemExactAsset(address _asset, uint256 _amount, uint256 _maxPoints) returns()
func (_Exchange *ExchangeTransactor) RedeemExactAsset(opts *bind.TransactOpts, _asset common.Address, _amount *big.Int, _maxPoints *big.Int) (*types.Transaction, error) {
	return _Exchange.contract.Transact(opts, "redeemExactAsset", _asset, _amount, _maxPoints)
}

// RedeemExactAsset is a paid mutator transaction binding the contract method 0x5b302d09.
//
// Solidity: function redeemExactAsset(address _asset, uint256 _amount, uint256 _maxPoints) returns()
func (_Exchange *ExchangeSession) RedeemExactAsset(_asset common.Address, _amount *big.Int, _maxPoints *big.Int) (*types.Transaction, error) {
	return _Exchange.Contract.RedeemExactAsset(&_Exchange.TransactOpts, _asset, _amount, _maxPoints)
}

// RedeemExactAsset is a paid mutator transaction binding the contract method 0x5b302d09.
//
// Solidity: function redeemExactAsset(address _asset, uint256 _amount, uint256 _maxPoints) returns()
func (_Exchange *ExchangeTransactorSession) RedeemExactAsset(_asset common.Address, _amount *big.Int, _maxPoints *big.Int) (*types.Transaction, error) {
	return _Exchange.Contract.RedeemExactAsset(&_Exchange.TransactOpts, _asset, _amount, _maxPoints)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Exchange *ExchangeTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Exchange.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Exchange *ExchangeSession) RenounceOwnership() (*types.Transaction, error) {
	return _Exchange.Contract.RenounceOwnership(&_Exchange.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Exchange *ExchangeTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _Exchange.Contract.RenounceOwnership(&_Exchange.TransactOpts)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Exchange *ExchangeTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _Exchange.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Exchange *ExchangeSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Exchange.Contract.TransferOwnership(&_Exchange.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Exchange *ExchangeTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Exchange.Contract.TransferOwnership(&_Exchange.TransactOpts, newOwner)
}

// UpdateStrategy is a paid mutator transaction binding the contract method 0xdaccaf63.
//
// Solidity: function updateStrategy(address _newStrategy) returns()
func (_Exchange *ExchangeTransactor) UpdateStrategy(opts *bind.TransactOpts, _newStrategy common.Address) (*types.Transaction, error) {
	return _Exchange.contract.Transact(opts, "updateStrategy", _newStrategy)
}

// UpdateStrategy is a paid mutator transaction binding the contract method 0xdaccaf63.
//
// Solidity: function updateStrategy(address _newStrategy) returns()
func (_Exchange *ExchangeSession) UpdateStrategy(_newStrategy common.Address) (*types.Transaction, error) {
	return _Exchange.Contract.UpdateStrategy(&_Exchange.TransactOpts, _newStrategy)
}

// UpdateStrategy is a paid mutator transaction binding the contract method 0xdaccaf63.
//
// Solidity: function updateStrategy(address _newStrategy) returns()
func (_Exchange *ExchangeTransactorSession) UpdateStrategy(_newStrategy common.Address) (*types.Transaction, error) {
	return _Exchange.Contract.UpdateStrategy(&_Exchange.TransactOpts, _newStrategy)
}

// ExchangeAssetAddedIterator is returned from FilterAssetAdded and is used to iterate over the raw logs and unpacked data for AssetAdded events raised by the Exchange contract.
type ExchangeAssetAddedIterator struct {
	Event *ExchangeAssetAdded // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ExchangeAssetAddedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ExchangeAssetAdded)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ExchangeAssetAdded)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ExchangeAssetAddedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ExchangeAssetAddedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ExchangeAssetAdded represents a AssetAdded event raised by the Exchange contract.
type ExchangeAssetAdded struct {
	Provider  common.Address
	Asset     common.Address
	Amount    *big.Int
	EndBlock  *big.Int
	Konstante *big.Int
	Raw       types.Log // Blockchain specific contextual infos
}

// FilterAssetAdded is a free log retrieval operation binding the contract event 0x0f0ce87d28aac0c07b026cb6025932cb4009646aeac7965d4cf463fdb1dd9ce0.
//
// Solidity: event AssetAdded(address indexed provider, address indexed asset, uint256 amount, uint256 endBlock, uint256 konstante)
func (_Exchange *ExchangeFilterer) FilterAssetAdded(opts *bind.FilterOpts, provider []common.Address, asset []common.Address) (*ExchangeAssetAddedIterator, error) {

	var providerRule []interface{}
	for _, providerItem := range provider {
		providerRule = append(providerRule, providerItem)
	}
	var assetRule []interface{}
	for _, assetItem := range asset {
		assetRule = append(assetRule, assetItem)
	}

	logs, sub, err := _Exchange.contract.FilterLogs(opts, "AssetAdded", providerRule, assetRule)
	if err != nil {
		return nil, err
	}
	return &ExchangeAssetAddedIterator{contract: _Exchange.contract, event: "AssetAdded", logs: logs, sub: sub}, nil
}

// WatchAssetAdded is a free log subscription operation binding the contract event 0x0f0ce87d28aac0c07b026cb6025932cb4009646aeac7965d4cf463fdb1dd9ce0.
//
// Solidity: event AssetAdded(address indexed provider, address indexed asset, uint256 amount, uint256 endBlock, uint256 konstante)
func (_Exchange *ExchangeFilterer) WatchAssetAdded(opts *bind.WatchOpts, sink chan<- *ExchangeAssetAdded, provider []common.Address, asset []common.Address) (event.Subscription, error) {

	var providerRule []interface{}
	for _, providerItem := range provider {
		providerRule = append(providerRule, providerItem)
	}
	var assetRule []interface{}
	for _, assetItem := range asset {
		assetRule = append(assetRule, assetItem)
	}

	logs, sub, err := _Exchange.contract.WatchLogs(opts, "AssetAdded", providerRule, assetRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ExchangeAssetAdded)
				if err := _Exchange.contract.UnpackLog(event, "AssetAdded", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseAssetAdded is a log parse operation binding the contract event 0x0f0ce87d28aac0c07b026cb6025932cb4009646aeac7965d4cf463fdb1dd9ce0.
//
// Solidity: event AssetAdded(address indexed provider, address indexed asset, uint256 amount, uint256 endBlock, uint256 konstante)
func (_Exchange *ExchangeFilterer) ParseAssetAdded(log types.Log) (*ExchangeAssetAdded, error) {
	event := new(ExchangeAssetAdded)
	if err := _Exchange.contract.UnpackLog(event, "AssetAdded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ExchangeAssetDrippedIterator is returned from FilterAssetDripped and is used to iterate over the raw logs and unpacked data for AssetDripped events raised by the Exchange contract.
type ExchangeAssetDrippedIterator struct {
	Event *ExchangeAssetDripped // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ExchangeAssetDrippedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ExchangeAssetDripped)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ExchangeAssetDripped)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ExchangeAssetDrippedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ExchangeAssetDrippedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ExchangeAssetDripped represents a AssetDripped event raised by the Exchange contract.
type ExchangeAssetDripped struct {
	Asset  common.Address
	Volume *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterAssetDripped is a free log retrieval operation binding the contract event 0xf616b63fa85386183d388f90cb6006662c309731f969461cd877ee055b306e3a.
//
// Solidity: event AssetDripped(address indexed asset, uint256 volume)
func (_Exchange *ExchangeFilterer) FilterAssetDripped(opts *bind.FilterOpts, asset []common.Address) (*ExchangeAssetDrippedIterator, error) {

	var assetRule []interface{}
	for _, assetItem := range asset {
		assetRule = append(assetRule, assetItem)
	}

	logs, sub, err := _Exchange.contract.FilterLogs(opts, "AssetDripped", assetRule)
	if err != nil {
		return nil, err
	}
	return &ExchangeAssetDrippedIterator{contract: _Exchange.contract, event: "AssetDripped", logs: logs, sub: sub}, nil
}

// WatchAssetDripped is a free log subscription operation binding the contract event 0xf616b63fa85386183d388f90cb6006662c309731f969461cd877ee055b306e3a.
//
// Solidity: event AssetDripped(address indexed asset, uint256 volume)
func (_Exchange *ExchangeFilterer) WatchAssetDripped(opts *bind.WatchOpts, sink chan<- *ExchangeAssetDripped, asset []common.Address) (event.Subscription, error) {

	var assetRule []interface{}
	for _, assetItem := range asset {
		assetRule = append(assetRule, assetItem)
	}

	logs, sub, err := _Exchange.contract.WatchLogs(opts, "AssetDripped", assetRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ExchangeAssetDripped)
				if err := _Exchange.contract.UnpackLog(event, "AssetDripped", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseAssetDripped is a log parse operation binding the contract event 0xf616b63fa85386183d388f90cb6006662c309731f969461cd877ee055b306e3a.
//
// Solidity: event AssetDripped(address indexed asset, uint256 volume)
func (_Exchange *ExchangeFilterer) ParseAssetDripped(log types.Log) (*ExchangeAssetDripped, error) {
	event := new(ExchangeAssetDripped)
	if err := _Exchange.contract.UnpackLog(event, "AssetDripped", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ExchangeOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the Exchange contract.
type ExchangeOwnershipTransferredIterator struct {
	Event *ExchangeOwnershipTransferred // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ExchangeOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ExchangeOwnershipTransferred)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ExchangeOwnershipTransferred)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ExchangeOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ExchangeOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ExchangeOwnershipTransferred represents a OwnershipTransferred event raised by the Exchange contract.
type ExchangeOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Exchange *ExchangeFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*ExchangeOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Exchange.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &ExchangeOwnershipTransferredIterator{contract: _Exchange.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Exchange *ExchangeFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *ExchangeOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Exchange.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ExchangeOwnershipTransferred)
				if err := _Exchange.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Exchange *ExchangeFilterer) ParseOwnershipTransferred(log types.Log) (*ExchangeOwnershipTransferred, error) {
	event := new(ExchangeOwnershipTransferred)
	if err := _Exchange.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ExchangeRedemptionIterator is returned from FilterRedemption and is used to iterate over the raw logs and unpacked data for Redemption events raised by the Exchange contract.
type ExchangeRedemptionIterator struct {
	Event *ExchangeRedemption // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ExchangeRedemptionIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ExchangeRedemption)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ExchangeRedemption)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ExchangeRedemptionIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ExchangeRedemptionIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ExchangeRedemption represents a Redemption event raised by the Exchange contract.
type ExchangeRedemption struct {
	User   common.Address
	Asset  common.Address
	Amount *big.Int
	Points *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterRedemption is a free log retrieval operation binding the contract event 0xa28d80c9910787c0c058ed9b50c577f1389264bf61563fa45529e0771976f562.
//
// Solidity: event Redemption(address indexed user, address indexed asset, uint256 amount, uint256 points)
func (_Exchange *ExchangeFilterer) FilterRedemption(opts *bind.FilterOpts, user []common.Address, asset []common.Address) (*ExchangeRedemptionIterator, error) {

	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}
	var assetRule []interface{}
	for _, assetItem := range asset {
		assetRule = append(assetRule, assetItem)
	}

	logs, sub, err := _Exchange.contract.FilterLogs(opts, "Redemption", userRule, assetRule)
	if err != nil {
		return nil, err
	}
	return &ExchangeRedemptionIterator{contract: _Exchange.contract, event: "Redemption", logs: logs, sub: sub}, nil
}

// WatchRedemption is a free log subscription operation binding the contract event 0xa28d80c9910787c0c058ed9b50c577f1389264bf61563fa45529e0771976f562.
//
// Solidity: event Redemption(address indexed user, address indexed asset, uint256 amount, uint256 points)
func (_Exchange *ExchangeFilterer) WatchRedemption(opts *bind.WatchOpts, sink chan<- *ExchangeRedemption, user []common.Address, asset []common.Address) (event.Subscription, error) {

	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}
	var assetRule []interface{}
	for _, assetItem := range asset {
		assetRule = append(assetRule, assetItem)
	}

	logs, sub, err := _Exchange.contract.WatchLogs(opts, "Redemption", userRule, assetRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ExchangeRedemption)
				if err := _Exchange.contract.UnpackLog(event, "Redemption", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseRedemption is a log parse operation binding the contract event 0xa28d80c9910787c0c058ed9b50c577f1389264bf61563fa45529e0771976f562.
//
// Solidity: event Redemption(address indexed user, address indexed asset, uint256 amount, uint256 points)
func (_Exchange *ExchangeFilterer) ParseRedemption(log types.Log) (*ExchangeRedemption, error) {
	event := new(ExchangeRedemption)
	if err := _Exchange.contract.UnpackLog(event, "Redemption", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// ExchangeStrategyUpdatedIterator is returned from FilterStrategyUpdated and is used to iterate over the raw logs and unpacked data for StrategyUpdated events raised by the Exchange contract.
type ExchangeStrategyUpdatedIterator struct {
	Event *ExchangeStrategyUpdated // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *ExchangeStrategyUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(ExchangeStrategyUpdated)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(ExchangeStrategyUpdated)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *ExchangeStrategyUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *ExchangeStrategyUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// ExchangeStrategyUpdated represents a StrategyUpdated event raised by the Exchange contract.
type ExchangeStrategyUpdated struct {
	Strategy common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterStrategyUpdated is a free log retrieval operation binding the contract event 0x4da9c22c924692646a21cf1f423781ae3285198dc22e8a6912835d3272b90b3c.
//
// Solidity: event StrategyUpdated(address indexed strategy)
func (_Exchange *ExchangeFilterer) FilterStrategyUpdated(opts *bind.FilterOpts, strategy []common.Address) (*ExchangeStrategyUpdatedIterator, error) {

	var strategyRule []interface{}
	for _, strategyItem := range strategy {
		strategyRule = append(strategyRule, strategyItem)
	}

	logs, sub, err := _Exchange.contract.FilterLogs(opts, "StrategyUpdated", strategyRule)
	if err != nil {
		return nil, err
	}
	return &ExchangeStrategyUpdatedIterator{contract: _Exchange.contract, event: "StrategyUpdated", logs: logs, sub: sub}, nil
}

// WatchStrategyUpdated is a free log subscription operation binding the contract event 0x4da9c22c924692646a21cf1f423781ae3285198dc22e8a6912835d3272b90b3c.
//
// Solidity: event StrategyUpdated(address indexed strategy)
func (_Exchange *ExchangeFilterer) WatchStrategyUpdated(opts *bind.WatchOpts, sink chan<- *ExchangeStrategyUpdated, strategy []common.Address) (event.Subscription, error) {

	var strategyRule []interface{}
	for _, strategyItem := range strategy {
		strategyRule = append(strategyRule, strategyItem)
	}

	logs, sub, err := _Exchange.contract.WatchLogs(opts, "StrategyUpdated", strategyRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(ExchangeStrategyUpdated)
				if err := _Exchange.contract.UnpackLog(event, "StrategyUpdated", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseStrategyUpdated is a log parse operation binding the contract event 0x4da9c22c924692646a21cf1f423781ae3285198dc22e8a6912835d3272b90b3c.
//
// Solidity: event StrategyUpdated(address indexed strategy)
func (_Exchange *ExchangeFilterer) ParseStrategyUpdated(log types.Log) (*ExchangeStrategyUpdated, error) {
	event := new(ExchangeStrategyUpdated)
	if err := _Exchange.contract.UnpackLog(event, "StrategyUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
