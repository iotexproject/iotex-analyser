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

// AccountantABI is the input ABI used to generate the binding from.
const AccountantABI = "[{\"constant\":false,\"inputs\":[{\"name\":\"_newAccountant\",\"type\":\"address\"}],\"name\":\"upgrade\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"register\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_user\",\"type\":\"address\"}],\"name\":\"claim\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"accountBook\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"period\",\"type\":\"uint256\"}],\"name\":\"updateValidPeriod\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"exchangeRate\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"withdraw\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"exchange\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_client\",\"type\":\"address\"}],\"name\":\"addTrustedClient\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"operator\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"accounts\",\"outputs\":[{\"name\":\"shares\",\"type\":\"uint256\"},{\"name\":\"debt\",\"type\":\"uint256\"},{\"name\":\"expireAt\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_user\",\"type\":\"address\"}],\"name\":\"balanceOf\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"validPeriod\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"termHeight\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"isOwner\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"term\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_account\",\"type\":\"address\"},{\"name\":\"_amount\",\"type\":\"uint256\"}],\"name\":\"charge\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_client\",\"type\":\"address\"}],\"name\":\"isTrustedClient\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_operator\",\"type\":\"address\"}],\"name\":\"setOperator\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_newRate\",\"type\":\"uint256\"}],\"name\":\"updateExchangeRate\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_client\",\"type\":\"address\"}],\"name\":\"removeTrustedClient\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_users\",\"type\":\"address[]\"},{\"name\":\"_shares\",\"type\":\"uint256[]\"}],\"name\":\"updateShares\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_height\",\"type\":\"uint256\"}],\"name\":\"nextTerm\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"token\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"_accountBook\",\"type\":\"address\"},{\"name\":\"_validPeriod\",\"type\":\"uint256\"},{\"name\":\"_token\",\"type\":\"address\"},{\"name\":\"_exchangeRate\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"user\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"expireAt\",\"type\":\"uint256\"}],\"name\":\"Registration\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"period\",\"type\":\"uint256\"}],\"name\":\"ValidPeriodUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"rate\",\"type\":\"uint256\"}],\"name\":\"ExchangeRateUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"user\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"amount\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"exchangeRate\",\"type\":\"uint256\"}],\"name\":\"Exchange\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"user\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"points\",\"type\":\"uint256\"}],\"name\":\"Claim\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"term\",\"type\":\"uint256\"},{\"indexed\":false,\"name\":\"termHeight\",\"type\":\"uint256\"}],\"name\":\"NewTerm\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"operator\",\"type\":\"address\"}],\"name\":\"OperatorSet\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"client\",\"type\":\"address\"}],\"name\":\"TrustedClientAdded\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"client\",\"type\":\"address\"}],\"name\":\"TrustedClientRemoved\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"}]"

// Accountant is an auto generated Go binding around an Ethereum contract.
type Accountant struct {
	AccountantCaller     // Read-only binding to the contract
	AccountantTransactor // Write-only binding to the contract
	AccountantFilterer   // Log filterer for contract events
}

// AccountantCaller is an auto generated read-only Go binding around an Ethereum contract.
type AccountantCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AccountantTransactor is an auto generated write-only Go binding around an Ethereum contract.
type AccountantTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AccountantFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type AccountantFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// AccountantSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type AccountantSession struct {
	Contract     *Accountant       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// AccountantCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type AccountantCallerSession struct {
	Contract *AccountantCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// AccountantTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type AccountantTransactorSession struct {
	Contract     *AccountantTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// AccountantRaw is an auto generated low-level Go binding around an Ethereum contract.
type AccountantRaw struct {
	Contract *Accountant // Generic contract binding to access the raw methods on
}

// AccountantCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type AccountantCallerRaw struct {
	Contract *AccountantCaller // Generic read-only contract binding to access the raw methods on
}

// AccountantTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type AccountantTransactorRaw struct {
	Contract *AccountantTransactor // Generic write-only contract binding to access the raw methods on
}

// NewAccountant creates a new instance of Accountant, bound to a specific deployed contract.
func NewAccountant(address common.Address, backend bind.ContractBackend) (*Accountant, error) {
	contract, err := bindAccountant(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Accountant{AccountantCaller: AccountantCaller{contract: contract}, AccountantTransactor: AccountantTransactor{contract: contract}, AccountantFilterer: AccountantFilterer{contract: contract}}, nil
}

// NewAccountantCaller creates a new read-only instance of Accountant, bound to a specific deployed contract.
func NewAccountantCaller(address common.Address, caller bind.ContractCaller) (*AccountantCaller, error) {
	contract, err := bindAccountant(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &AccountantCaller{contract: contract}, nil
}

// NewAccountantTransactor creates a new write-only instance of Accountant, bound to a specific deployed contract.
func NewAccountantTransactor(address common.Address, transactor bind.ContractTransactor) (*AccountantTransactor, error) {
	contract, err := bindAccountant(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &AccountantTransactor{contract: contract}, nil
}

// NewAccountantFilterer creates a new log filterer instance of Accountant, bound to a specific deployed contract.
func NewAccountantFilterer(address common.Address, filterer bind.ContractFilterer) (*AccountantFilterer, error) {
	contract, err := bindAccountant(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &AccountantFilterer{contract: contract}, nil
}

// bindAccountant binds a generic wrapper to an already deployed contract.
func bindAccountant(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(AccountantABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Accountant *AccountantRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Accountant.Contract.AccountantCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Accountant *AccountantRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Accountant.Contract.AccountantTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Accountant *AccountantRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Accountant.Contract.AccountantTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Accountant *AccountantCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Accountant.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Accountant *AccountantTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Accountant.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Accountant *AccountantTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Accountant.Contract.contract.Transact(opts, method, params...)
}

// AccountBook is a free data retrieval call binding the contract method 0x2495bdb3.
//
// Solidity: function accountBook() view returns(address)
func (_Accountant *AccountantCaller) AccountBook(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Accountant.contract.Call(opts, &out, "accountBook")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// AccountBook is a free data retrieval call binding the contract method 0x2495bdb3.
//
// Solidity: function accountBook() view returns(address)
func (_Accountant *AccountantSession) AccountBook() (common.Address, error) {
	return _Accountant.Contract.AccountBook(&_Accountant.CallOpts)
}

// AccountBook is a free data retrieval call binding the contract method 0x2495bdb3.
//
// Solidity: function accountBook() view returns(address)
func (_Accountant *AccountantCallerSession) AccountBook() (common.Address, error) {
	return _Accountant.Contract.AccountBook(&_Accountant.CallOpts)
}

// Accounts is a free data retrieval call binding the contract method 0x5e5c06e2.
//
// Solidity: function accounts(address ) view returns(uint256 shares, uint256 debt, uint256 expireAt)
func (_Accountant *AccountantCaller) Accounts(opts *bind.CallOpts, arg0 common.Address) (struct {
	Shares   *big.Int
	Debt     *big.Int
	ExpireAt *big.Int
}, error) {
	var out []interface{}
	err := _Accountant.contract.Call(opts, &out, "accounts", arg0)

	outstruct := new(struct {
		Shares   *big.Int
		Debt     *big.Int
		ExpireAt *big.Int
	})

	outstruct.Shares = out[0].(*big.Int)
	outstruct.Debt = out[1].(*big.Int)
	outstruct.ExpireAt = out[2].(*big.Int)

	return *outstruct, err

}

// Accounts is a free data retrieval call binding the contract method 0x5e5c06e2.
//
// Solidity: function accounts(address ) view returns(uint256 shares, uint256 debt, uint256 expireAt)
func (_Accountant *AccountantSession) Accounts(arg0 common.Address) (struct {
	Shares   *big.Int
	Debt     *big.Int
	ExpireAt *big.Int
}, error) {
	return _Accountant.Contract.Accounts(&_Accountant.CallOpts, arg0)
}

// Accounts is a free data retrieval call binding the contract method 0x5e5c06e2.
//
// Solidity: function accounts(address ) view returns(uint256 shares, uint256 debt, uint256 expireAt)
func (_Accountant *AccountantCallerSession) Accounts(arg0 common.Address) (struct {
	Shares   *big.Int
	Debt     *big.Int
	ExpireAt *big.Int
}, error) {
	return _Accountant.Contract.Accounts(&_Accountant.CallOpts, arg0)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address _user) view returns(uint256)
func (_Accountant *AccountantCaller) BalanceOf(opts *bind.CallOpts, _user common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Accountant.contract.Call(opts, &out, "balanceOf", _user)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address _user) view returns(uint256)
func (_Accountant *AccountantSession) BalanceOf(_user common.Address) (*big.Int, error) {
	return _Accountant.Contract.BalanceOf(&_Accountant.CallOpts, _user)
}

// BalanceOf is a free data retrieval call binding the contract method 0x70a08231.
//
// Solidity: function balanceOf(address _user) view returns(uint256)
func (_Accountant *AccountantCallerSession) BalanceOf(_user common.Address) (*big.Int, error) {
	return _Accountant.Contract.BalanceOf(&_Accountant.CallOpts, _user)
}

// ExchangeRate is a free data retrieval call binding the contract method 0x3ba0b9a9.
//
// Solidity: function exchangeRate() view returns(uint256)
func (_Accountant *AccountantCaller) ExchangeRate(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Accountant.contract.Call(opts, &out, "exchangeRate")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ExchangeRate is a free data retrieval call binding the contract method 0x3ba0b9a9.
//
// Solidity: function exchangeRate() view returns(uint256)
func (_Accountant *AccountantSession) ExchangeRate() (*big.Int, error) {
	return _Accountant.Contract.ExchangeRate(&_Accountant.CallOpts)
}

// ExchangeRate is a free data retrieval call binding the contract method 0x3ba0b9a9.
//
// Solidity: function exchangeRate() view returns(uint256)
func (_Accountant *AccountantCallerSession) ExchangeRate() (*big.Int, error) {
	return _Accountant.Contract.ExchangeRate(&_Accountant.CallOpts)
}

// IsOwner is a free data retrieval call binding the contract method 0x8f32d59b.
//
// Solidity: function isOwner() view returns(bool)
func (_Accountant *AccountantCaller) IsOwner(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _Accountant.contract.Call(opts, &out, "isOwner")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsOwner is a free data retrieval call binding the contract method 0x8f32d59b.
//
// Solidity: function isOwner() view returns(bool)
func (_Accountant *AccountantSession) IsOwner() (bool, error) {
	return _Accountant.Contract.IsOwner(&_Accountant.CallOpts)
}

// IsOwner is a free data retrieval call binding the contract method 0x8f32d59b.
//
// Solidity: function isOwner() view returns(bool)
func (_Accountant *AccountantCallerSession) IsOwner() (bool, error) {
	return _Accountant.Contract.IsOwner(&_Accountant.CallOpts)
}

// IsTrustedClient is a free data retrieval call binding the contract method 0xb0d7b249.
//
// Solidity: function isTrustedClient(address _client) view returns(bool)
func (_Accountant *AccountantCaller) IsTrustedClient(opts *bind.CallOpts, _client common.Address) (bool, error) {
	var out []interface{}
	err := _Accountant.contract.Call(opts, &out, "isTrustedClient", _client)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsTrustedClient is a free data retrieval call binding the contract method 0xb0d7b249.
//
// Solidity: function isTrustedClient(address _client) view returns(bool)
func (_Accountant *AccountantSession) IsTrustedClient(_client common.Address) (bool, error) {
	return _Accountant.Contract.IsTrustedClient(&_Accountant.CallOpts, _client)
}

// IsTrustedClient is a free data retrieval call binding the contract method 0xb0d7b249.
//
// Solidity: function isTrustedClient(address _client) view returns(bool)
func (_Accountant *AccountantCallerSession) IsTrustedClient(_client common.Address) (bool, error) {
	return _Accountant.Contract.IsTrustedClient(&_Accountant.CallOpts, _client)
}

// Operator is a free data retrieval call binding the contract method 0x570ca735.
//
// Solidity: function operator() view returns(address)
func (_Accountant *AccountantCaller) Operator(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Accountant.contract.Call(opts, &out, "operator")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Operator is a free data retrieval call binding the contract method 0x570ca735.
//
// Solidity: function operator() view returns(address)
func (_Accountant *AccountantSession) Operator() (common.Address, error) {
	return _Accountant.Contract.Operator(&_Accountant.CallOpts)
}

// Operator is a free data retrieval call binding the contract method 0x570ca735.
//
// Solidity: function operator() view returns(address)
func (_Accountant *AccountantCallerSession) Operator() (common.Address, error) {
	return _Accountant.Contract.Operator(&_Accountant.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Accountant *AccountantCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Accountant.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Accountant *AccountantSession) Owner() (common.Address, error) {
	return _Accountant.Contract.Owner(&_Accountant.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Accountant *AccountantCallerSession) Owner() (common.Address, error) {
	return _Accountant.Contract.Owner(&_Accountant.CallOpts)
}

// Term is a free data retrieval call binding the contract method 0xa10ffbed.
//
// Solidity: function term() view returns(uint256)
func (_Accountant *AccountantCaller) Term(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Accountant.contract.Call(opts, &out, "term")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Term is a free data retrieval call binding the contract method 0xa10ffbed.
//
// Solidity: function term() view returns(uint256)
func (_Accountant *AccountantSession) Term() (*big.Int, error) {
	return _Accountant.Contract.Term(&_Accountant.CallOpts)
}

// Term is a free data retrieval call binding the contract method 0xa10ffbed.
//
// Solidity: function term() view returns(uint256)
func (_Accountant *AccountantCallerSession) Term() (*big.Int, error) {
	return _Accountant.Contract.Term(&_Accountant.CallOpts)
}

// TermHeight is a free data retrieval call binding the contract method 0x8cbae167.
//
// Solidity: function termHeight() view returns(uint256)
func (_Accountant *AccountantCaller) TermHeight(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Accountant.contract.Call(opts, &out, "termHeight")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// TermHeight is a free data retrieval call binding the contract method 0x8cbae167.
//
// Solidity: function termHeight() view returns(uint256)
func (_Accountant *AccountantSession) TermHeight() (*big.Int, error) {
	return _Accountant.Contract.TermHeight(&_Accountant.CallOpts)
}

// TermHeight is a free data retrieval call binding the contract method 0x8cbae167.
//
// Solidity: function termHeight() view returns(uint256)
func (_Accountant *AccountantCallerSession) TermHeight() (*big.Int, error) {
	return _Accountant.Contract.TermHeight(&_Accountant.CallOpts)
}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() view returns(address)
func (_Accountant *AccountantCaller) Token(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Accountant.contract.Call(opts, &out, "token")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() view returns(address)
func (_Accountant *AccountantSession) Token() (common.Address, error) {
	return _Accountant.Contract.Token(&_Accountant.CallOpts)
}

// Token is a free data retrieval call binding the contract method 0xfc0c546a.
//
// Solidity: function token() view returns(address)
func (_Accountant *AccountantCallerSession) Token() (common.Address, error) {
	return _Accountant.Contract.Token(&_Accountant.CallOpts)
}

// ValidPeriod is a free data retrieval call binding the contract method 0x7efcc3a2.
//
// Solidity: function validPeriod() view returns(uint256)
func (_Accountant *AccountantCaller) ValidPeriod(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Accountant.contract.Call(opts, &out, "validPeriod")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// ValidPeriod is a free data retrieval call binding the contract method 0x7efcc3a2.
//
// Solidity: function validPeriod() view returns(uint256)
func (_Accountant *AccountantSession) ValidPeriod() (*big.Int, error) {
	return _Accountant.Contract.ValidPeriod(&_Accountant.CallOpts)
}

// ValidPeriod is a free data retrieval call binding the contract method 0x7efcc3a2.
//
// Solidity: function validPeriod() view returns(uint256)
func (_Accountant *AccountantCallerSession) ValidPeriod() (*big.Int, error) {
	return _Accountant.Contract.ValidPeriod(&_Accountant.CallOpts)
}

// AddTrustedClient is a paid mutator transaction binding the contract method 0x55d9d47b.
//
// Solidity: function addTrustedClient(address _client) returns()
func (_Accountant *AccountantTransactor) AddTrustedClient(opts *bind.TransactOpts, _client common.Address) (*types.Transaction, error) {
	return _Accountant.contract.Transact(opts, "addTrustedClient", _client)
}

// AddTrustedClient is a paid mutator transaction binding the contract method 0x55d9d47b.
//
// Solidity: function addTrustedClient(address _client) returns()
func (_Accountant *AccountantSession) AddTrustedClient(_client common.Address) (*types.Transaction, error) {
	return _Accountant.Contract.AddTrustedClient(&_Accountant.TransactOpts, _client)
}

// AddTrustedClient is a paid mutator transaction binding the contract method 0x55d9d47b.
//
// Solidity: function addTrustedClient(address _client) returns()
func (_Accountant *AccountantTransactorSession) AddTrustedClient(_client common.Address) (*types.Transaction, error) {
	return _Accountant.Contract.AddTrustedClient(&_Accountant.TransactOpts, _client)
}

// Charge is a paid mutator transaction binding the contract method 0xa3ffa9cd.
//
// Solidity: function charge(address _account, uint256 _amount) returns()
func (_Accountant *AccountantTransactor) Charge(opts *bind.TransactOpts, _account common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _Accountant.contract.Transact(opts, "charge", _account, _amount)
}

// Charge is a paid mutator transaction binding the contract method 0xa3ffa9cd.
//
// Solidity: function charge(address _account, uint256 _amount) returns()
func (_Accountant *AccountantSession) Charge(_account common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _Accountant.Contract.Charge(&_Accountant.TransactOpts, _account, _amount)
}

// Charge is a paid mutator transaction binding the contract method 0xa3ffa9cd.
//
// Solidity: function charge(address _account, uint256 _amount) returns()
func (_Accountant *AccountantTransactorSession) Charge(_account common.Address, _amount *big.Int) (*types.Transaction, error) {
	return _Accountant.Contract.Charge(&_Accountant.TransactOpts, _account, _amount)
}

// Claim is a paid mutator transaction binding the contract method 0x1e83409a.
//
// Solidity: function claim(address _user) returns()
func (_Accountant *AccountantTransactor) Claim(opts *bind.TransactOpts, _user common.Address) (*types.Transaction, error) {
	return _Accountant.contract.Transact(opts, "claim", _user)
}

// Claim is a paid mutator transaction binding the contract method 0x1e83409a.
//
// Solidity: function claim(address _user) returns()
func (_Accountant *AccountantSession) Claim(_user common.Address) (*types.Transaction, error) {
	return _Accountant.Contract.Claim(&_Accountant.TransactOpts, _user)
}

// Claim is a paid mutator transaction binding the contract method 0x1e83409a.
//
// Solidity: function claim(address _user) returns()
func (_Accountant *AccountantTransactorSession) Claim(_user common.Address) (*types.Transaction, error) {
	return _Accountant.Contract.Claim(&_Accountant.TransactOpts, _user)
}

// Exchange is a paid mutator transaction binding the contract method 0x53556559.
//
// Solidity: function exchange(uint256 _amount) returns()
func (_Accountant *AccountantTransactor) Exchange(opts *bind.TransactOpts, _amount *big.Int) (*types.Transaction, error) {
	return _Accountant.contract.Transact(opts, "exchange", _amount)
}

// Exchange is a paid mutator transaction binding the contract method 0x53556559.
//
// Solidity: function exchange(uint256 _amount) returns()
func (_Accountant *AccountantSession) Exchange(_amount *big.Int) (*types.Transaction, error) {
	return _Accountant.Contract.Exchange(&_Accountant.TransactOpts, _amount)
}

// Exchange is a paid mutator transaction binding the contract method 0x53556559.
//
// Solidity: function exchange(uint256 _amount) returns()
func (_Accountant *AccountantTransactorSession) Exchange(_amount *big.Int) (*types.Transaction, error) {
	return _Accountant.Contract.Exchange(&_Accountant.TransactOpts, _amount)
}

// NextTerm is a paid mutator transaction binding the contract method 0xf155a9bb.
//
// Solidity: function nextTerm(uint256 _height) returns()
func (_Accountant *AccountantTransactor) NextTerm(opts *bind.TransactOpts, _height *big.Int) (*types.Transaction, error) {
	return _Accountant.contract.Transact(opts, "nextTerm", _height)
}

// NextTerm is a paid mutator transaction binding the contract method 0xf155a9bb.
//
// Solidity: function nextTerm(uint256 _height) returns()
func (_Accountant *AccountantSession) NextTerm(_height *big.Int) (*types.Transaction, error) {
	return _Accountant.Contract.NextTerm(&_Accountant.TransactOpts, _height)
}

// NextTerm is a paid mutator transaction binding the contract method 0xf155a9bb.
//
// Solidity: function nextTerm(uint256 _height) returns()
func (_Accountant *AccountantTransactorSession) NextTerm(_height *big.Int) (*types.Transaction, error) {
	return _Accountant.Contract.NextTerm(&_Accountant.TransactOpts, _height)
}

// Register is a paid mutator transaction binding the contract method 0x1aa3a008.
//
// Solidity: function register() returns()
func (_Accountant *AccountantTransactor) Register(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Accountant.contract.Transact(opts, "register")
}

// Register is a paid mutator transaction binding the contract method 0x1aa3a008.
//
// Solidity: function register() returns()
func (_Accountant *AccountantSession) Register() (*types.Transaction, error) {
	return _Accountant.Contract.Register(&_Accountant.TransactOpts)
}

// Register is a paid mutator transaction binding the contract method 0x1aa3a008.
//
// Solidity: function register() returns()
func (_Accountant *AccountantTransactorSession) Register() (*types.Transaction, error) {
	return _Accountant.Contract.Register(&_Accountant.TransactOpts)
}

// RemoveTrustedClient is a paid mutator transaction binding the contract method 0xc2e89479.
//
// Solidity: function removeTrustedClient(address _client) returns()
func (_Accountant *AccountantTransactor) RemoveTrustedClient(opts *bind.TransactOpts, _client common.Address) (*types.Transaction, error) {
	return _Accountant.contract.Transact(opts, "removeTrustedClient", _client)
}

// RemoveTrustedClient is a paid mutator transaction binding the contract method 0xc2e89479.
//
// Solidity: function removeTrustedClient(address _client) returns()
func (_Accountant *AccountantSession) RemoveTrustedClient(_client common.Address) (*types.Transaction, error) {
	return _Accountant.Contract.RemoveTrustedClient(&_Accountant.TransactOpts, _client)
}

// RemoveTrustedClient is a paid mutator transaction binding the contract method 0xc2e89479.
//
// Solidity: function removeTrustedClient(address _client) returns()
func (_Accountant *AccountantTransactorSession) RemoveTrustedClient(_client common.Address) (*types.Transaction, error) {
	return _Accountant.Contract.RemoveTrustedClient(&_Accountant.TransactOpts, _client)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Accountant *AccountantTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Accountant.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Accountant *AccountantSession) RenounceOwnership() (*types.Transaction, error) {
	return _Accountant.Contract.RenounceOwnership(&_Accountant.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_Accountant *AccountantTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _Accountant.Contract.RenounceOwnership(&_Accountant.TransactOpts)
}

// SetOperator is a paid mutator transaction binding the contract method 0xb3ab15fb.
//
// Solidity: function setOperator(address _operator) returns()
func (_Accountant *AccountantTransactor) SetOperator(opts *bind.TransactOpts, _operator common.Address) (*types.Transaction, error) {
	return _Accountant.contract.Transact(opts, "setOperator", _operator)
}

// SetOperator is a paid mutator transaction binding the contract method 0xb3ab15fb.
//
// Solidity: function setOperator(address _operator) returns()
func (_Accountant *AccountantSession) SetOperator(_operator common.Address) (*types.Transaction, error) {
	return _Accountant.Contract.SetOperator(&_Accountant.TransactOpts, _operator)
}

// SetOperator is a paid mutator transaction binding the contract method 0xb3ab15fb.
//
// Solidity: function setOperator(address _operator) returns()
func (_Accountant *AccountantTransactorSession) SetOperator(_operator common.Address) (*types.Transaction, error) {
	return _Accountant.Contract.SetOperator(&_Accountant.TransactOpts, _operator)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Accountant *AccountantTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _Accountant.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Accountant *AccountantSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Accountant.Contract.TransferOwnership(&_Accountant.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_Accountant *AccountantTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _Accountant.Contract.TransferOwnership(&_Accountant.TransactOpts, newOwner)
}

// UpdateExchangeRate is a paid mutator transaction binding the contract method 0xb9e205ae.
//
// Solidity: function updateExchangeRate(uint256 _newRate) returns()
func (_Accountant *AccountantTransactor) UpdateExchangeRate(opts *bind.TransactOpts, _newRate *big.Int) (*types.Transaction, error) {
	return _Accountant.contract.Transact(opts, "updateExchangeRate", _newRate)
}

// UpdateExchangeRate is a paid mutator transaction binding the contract method 0xb9e205ae.
//
// Solidity: function updateExchangeRate(uint256 _newRate) returns()
func (_Accountant *AccountantSession) UpdateExchangeRate(_newRate *big.Int) (*types.Transaction, error) {
	return _Accountant.Contract.UpdateExchangeRate(&_Accountant.TransactOpts, _newRate)
}

// UpdateExchangeRate is a paid mutator transaction binding the contract method 0xb9e205ae.
//
// Solidity: function updateExchangeRate(uint256 _newRate) returns()
func (_Accountant *AccountantTransactorSession) UpdateExchangeRate(_newRate *big.Int) (*types.Transaction, error) {
	return _Accountant.Contract.UpdateExchangeRate(&_Accountant.TransactOpts, _newRate)
}

// UpdateShares is a paid mutator transaction binding the contract method 0xe5bc026c.
//
// Solidity: function updateShares(address[] _users, uint256[] _shares) returns()
func (_Accountant *AccountantTransactor) UpdateShares(opts *bind.TransactOpts, _users []common.Address, _shares []*big.Int) (*types.Transaction, error) {
	return _Accountant.contract.Transact(opts, "updateShares", _users, _shares)
}

// UpdateShares is a paid mutator transaction binding the contract method 0xe5bc026c.
//
// Solidity: function updateShares(address[] _users, uint256[] _shares) returns()
func (_Accountant *AccountantSession) UpdateShares(_users []common.Address, _shares []*big.Int) (*types.Transaction, error) {
	return _Accountant.Contract.UpdateShares(&_Accountant.TransactOpts, _users, _shares)
}

// UpdateShares is a paid mutator transaction binding the contract method 0xe5bc026c.
//
// Solidity: function updateShares(address[] _users, uint256[] _shares) returns()
func (_Accountant *AccountantTransactorSession) UpdateShares(_users []common.Address, _shares []*big.Int) (*types.Transaction, error) {
	return _Accountant.Contract.UpdateShares(&_Accountant.TransactOpts, _users, _shares)
}

// UpdateValidPeriod is a paid mutator transaction binding the contract method 0x36e265ad.
//
// Solidity: function updateValidPeriod(uint256 period) returns()
func (_Accountant *AccountantTransactor) UpdateValidPeriod(opts *bind.TransactOpts, period *big.Int) (*types.Transaction, error) {
	return _Accountant.contract.Transact(opts, "updateValidPeriod", period)
}

// UpdateValidPeriod is a paid mutator transaction binding the contract method 0x36e265ad.
//
// Solidity: function updateValidPeriod(uint256 period) returns()
func (_Accountant *AccountantSession) UpdateValidPeriod(period *big.Int) (*types.Transaction, error) {
	return _Accountant.Contract.UpdateValidPeriod(&_Accountant.TransactOpts, period)
}

// UpdateValidPeriod is a paid mutator transaction binding the contract method 0x36e265ad.
//
// Solidity: function updateValidPeriod(uint256 period) returns()
func (_Accountant *AccountantTransactorSession) UpdateValidPeriod(period *big.Int) (*types.Transaction, error) {
	return _Accountant.Contract.UpdateValidPeriod(&_Accountant.TransactOpts, period)
}

// Upgrade is a paid mutator transaction binding the contract method 0x0900f010.
//
// Solidity: function upgrade(address _newAccountant) returns()
func (_Accountant *AccountantTransactor) Upgrade(opts *bind.TransactOpts, _newAccountant common.Address) (*types.Transaction, error) {
	return _Accountant.contract.Transact(opts, "upgrade", _newAccountant)
}

// Upgrade is a paid mutator transaction binding the contract method 0x0900f010.
//
// Solidity: function upgrade(address _newAccountant) returns()
func (_Accountant *AccountantSession) Upgrade(_newAccountant common.Address) (*types.Transaction, error) {
	return _Accountant.Contract.Upgrade(&_Accountant.TransactOpts, _newAccountant)
}

// Upgrade is a paid mutator transaction binding the contract method 0x0900f010.
//
// Solidity: function upgrade(address _newAccountant) returns()
func (_Accountant *AccountantTransactorSession) Upgrade(_newAccountant common.Address) (*types.Transaction, error) {
	return _Accountant.Contract.Upgrade(&_Accountant.TransactOpts, _newAccountant)
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_Accountant *AccountantTransactor) Withdraw(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Accountant.contract.Transact(opts, "withdraw")
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_Accountant *AccountantSession) Withdraw() (*types.Transaction, error) {
	return _Accountant.Contract.Withdraw(&_Accountant.TransactOpts)
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_Accountant *AccountantTransactorSession) Withdraw() (*types.Transaction, error) {
	return _Accountant.Contract.Withdraw(&_Accountant.TransactOpts)
}

// AccountantClaimIterator is returned from FilterClaim and is used to iterate over the raw logs and unpacked data for Claim events raised by the Accountant contract.
type AccountantClaimIterator struct {
	Event *AccountantClaim // Event containing the contract specifics and raw log

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
func (it *AccountantClaimIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AccountantClaim)
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
		it.Event = new(AccountantClaim)
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
func (it *AccountantClaimIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AccountantClaimIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AccountantClaim represents a Claim event raised by the Accountant contract.
type AccountantClaim struct {
	User   common.Address
	Points *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterClaim is a free log retrieval operation binding the contract event 0x47cee97cb7acd717b3c0aa1435d004cd5b3c8c57d70dbceb4e4458bbd60e39d4.
//
// Solidity: event Claim(address indexed user, uint256 points)
func (_Accountant *AccountantFilterer) FilterClaim(opts *bind.FilterOpts, user []common.Address) (*AccountantClaimIterator, error) {

	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	logs, sub, err := _Accountant.contract.FilterLogs(opts, "Claim", userRule)
	if err != nil {
		return nil, err
	}
	return &AccountantClaimIterator{contract: _Accountant.contract, event: "Claim", logs: logs, sub: sub}, nil
}

// WatchClaim is a free log subscription operation binding the contract event 0x47cee97cb7acd717b3c0aa1435d004cd5b3c8c57d70dbceb4e4458bbd60e39d4.
//
// Solidity: event Claim(address indexed user, uint256 points)
func (_Accountant *AccountantFilterer) WatchClaim(opts *bind.WatchOpts, sink chan<- *AccountantClaim, user []common.Address) (event.Subscription, error) {

	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	logs, sub, err := _Accountant.contract.WatchLogs(opts, "Claim", userRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AccountantClaim)
				if err := _Accountant.contract.UnpackLog(event, "Claim", log); err != nil {
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

// ParseClaim is a log parse operation binding the contract event 0x47cee97cb7acd717b3c0aa1435d004cd5b3c8c57d70dbceb4e4458bbd60e39d4.
//
// Solidity: event Claim(address indexed user, uint256 points)
func (_Accountant *AccountantFilterer) ParseClaim(log types.Log) (*AccountantClaim, error) {
	event := new(AccountantClaim)
	if err := _Accountant.contract.UnpackLog(event, "Claim", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AccountantExchangeIterator is returned from FilterExchange and is used to iterate over the raw logs and unpacked data for Exchange events raised by the Accountant contract.
type AccountantExchangeIterator struct {
	Event *AccountantExchange // Event containing the contract specifics and raw log

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
func (it *AccountantExchangeIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AccountantExchange)
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
		it.Event = new(AccountantExchange)
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
func (it *AccountantExchangeIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AccountantExchangeIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AccountantExchange represents a Exchange event raised by the Accountant contract.
type AccountantExchange struct {
	User         common.Address
	Amount       *big.Int
	ExchangeRate *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterExchange is a free log retrieval operation binding the contract event 0x26981b9aefbb0f732b0264bd34c255e831001eb50b06bc85b32cc39e14389721.
//
// Solidity: event Exchange(address indexed user, uint256 amount, uint256 exchangeRate)
func (_Accountant *AccountantFilterer) FilterExchange(opts *bind.FilterOpts, user []common.Address) (*AccountantExchangeIterator, error) {

	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	logs, sub, err := _Accountant.contract.FilterLogs(opts, "Exchange", userRule)
	if err != nil {
		return nil, err
	}
	return &AccountantExchangeIterator{contract: _Accountant.contract, event: "Exchange", logs: logs, sub: sub}, nil
}

// WatchExchange is a free log subscription operation binding the contract event 0x26981b9aefbb0f732b0264bd34c255e831001eb50b06bc85b32cc39e14389721.
//
// Solidity: event Exchange(address indexed user, uint256 amount, uint256 exchangeRate)
func (_Accountant *AccountantFilterer) WatchExchange(opts *bind.WatchOpts, sink chan<- *AccountantExchange, user []common.Address) (event.Subscription, error) {

	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	logs, sub, err := _Accountant.contract.WatchLogs(opts, "Exchange", userRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AccountantExchange)
				if err := _Accountant.contract.UnpackLog(event, "Exchange", log); err != nil {
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

// ParseExchange is a log parse operation binding the contract event 0x26981b9aefbb0f732b0264bd34c255e831001eb50b06bc85b32cc39e14389721.
//
// Solidity: event Exchange(address indexed user, uint256 amount, uint256 exchangeRate)
func (_Accountant *AccountantFilterer) ParseExchange(log types.Log) (*AccountantExchange, error) {
	event := new(AccountantExchange)
	if err := _Accountant.contract.UnpackLog(event, "Exchange", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AccountantExchangeRateUpdatedIterator is returned from FilterExchangeRateUpdated and is used to iterate over the raw logs and unpacked data for ExchangeRateUpdated events raised by the Accountant contract.
type AccountantExchangeRateUpdatedIterator struct {
	Event *AccountantExchangeRateUpdated // Event containing the contract specifics and raw log

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
func (it *AccountantExchangeRateUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AccountantExchangeRateUpdated)
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
		it.Event = new(AccountantExchangeRateUpdated)
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
func (it *AccountantExchangeRateUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AccountantExchangeRateUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AccountantExchangeRateUpdated represents a ExchangeRateUpdated event raised by the Accountant contract.
type AccountantExchangeRateUpdated struct {
	Rate *big.Int
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterExchangeRateUpdated is a free log retrieval operation binding the contract event 0x388f446e9526fe5c9af20a5919b342370c8a7c0cb05245afe1e545658fa3cdba.
//
// Solidity: event ExchangeRateUpdated(uint256 rate)
func (_Accountant *AccountantFilterer) FilterExchangeRateUpdated(opts *bind.FilterOpts) (*AccountantExchangeRateUpdatedIterator, error) {

	logs, sub, err := _Accountant.contract.FilterLogs(opts, "ExchangeRateUpdated")
	if err != nil {
		return nil, err
	}
	return &AccountantExchangeRateUpdatedIterator{contract: _Accountant.contract, event: "ExchangeRateUpdated", logs: logs, sub: sub}, nil
}

// WatchExchangeRateUpdated is a free log subscription operation binding the contract event 0x388f446e9526fe5c9af20a5919b342370c8a7c0cb05245afe1e545658fa3cdba.
//
// Solidity: event ExchangeRateUpdated(uint256 rate)
func (_Accountant *AccountantFilterer) WatchExchangeRateUpdated(opts *bind.WatchOpts, sink chan<- *AccountantExchangeRateUpdated) (event.Subscription, error) {

	logs, sub, err := _Accountant.contract.WatchLogs(opts, "ExchangeRateUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AccountantExchangeRateUpdated)
				if err := _Accountant.contract.UnpackLog(event, "ExchangeRateUpdated", log); err != nil {
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

// ParseExchangeRateUpdated is a log parse operation binding the contract event 0x388f446e9526fe5c9af20a5919b342370c8a7c0cb05245afe1e545658fa3cdba.
//
// Solidity: event ExchangeRateUpdated(uint256 rate)
func (_Accountant *AccountantFilterer) ParseExchangeRateUpdated(log types.Log) (*AccountantExchangeRateUpdated, error) {
	event := new(AccountantExchangeRateUpdated)
	if err := _Accountant.contract.UnpackLog(event, "ExchangeRateUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AccountantNewTermIterator is returned from FilterNewTerm and is used to iterate over the raw logs and unpacked data for NewTerm events raised by the Accountant contract.
type AccountantNewTermIterator struct {
	Event *AccountantNewTerm // Event containing the contract specifics and raw log

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
func (it *AccountantNewTermIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AccountantNewTerm)
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
		it.Event = new(AccountantNewTerm)
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
func (it *AccountantNewTermIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AccountantNewTermIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AccountantNewTerm represents a NewTerm event raised by the Accountant contract.
type AccountantNewTerm struct {
	Term       *big.Int
	TermHeight *big.Int
	Raw        types.Log // Blockchain specific contextual infos
}

// FilterNewTerm is a free log retrieval operation binding the contract event 0x98f0d489a1bb3ac90294374182d5b02db885776732c344172c51f27802be6531.
//
// Solidity: event NewTerm(uint256 indexed term, uint256 termHeight)
func (_Accountant *AccountantFilterer) FilterNewTerm(opts *bind.FilterOpts, term []*big.Int) (*AccountantNewTermIterator, error) {

	var termRule []interface{}
	for _, termItem := range term {
		termRule = append(termRule, termItem)
	}

	logs, sub, err := _Accountant.contract.FilterLogs(opts, "NewTerm", termRule)
	if err != nil {
		return nil, err
	}
	return &AccountantNewTermIterator{contract: _Accountant.contract, event: "NewTerm", logs: logs, sub: sub}, nil
}

// WatchNewTerm is a free log subscription operation binding the contract event 0x98f0d489a1bb3ac90294374182d5b02db885776732c344172c51f27802be6531.
//
// Solidity: event NewTerm(uint256 indexed term, uint256 termHeight)
func (_Accountant *AccountantFilterer) WatchNewTerm(opts *bind.WatchOpts, sink chan<- *AccountantNewTerm, term []*big.Int) (event.Subscription, error) {

	var termRule []interface{}
	for _, termItem := range term {
		termRule = append(termRule, termItem)
	}

	logs, sub, err := _Accountant.contract.WatchLogs(opts, "NewTerm", termRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AccountantNewTerm)
				if err := _Accountant.contract.UnpackLog(event, "NewTerm", log); err != nil {
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

// ParseNewTerm is a log parse operation binding the contract event 0x98f0d489a1bb3ac90294374182d5b02db885776732c344172c51f27802be6531.
//
// Solidity: event NewTerm(uint256 indexed term, uint256 termHeight)
func (_Accountant *AccountantFilterer) ParseNewTerm(log types.Log) (*AccountantNewTerm, error) {
	event := new(AccountantNewTerm)
	if err := _Accountant.contract.UnpackLog(event, "NewTerm", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AccountantOperatorSetIterator is returned from FilterOperatorSet and is used to iterate over the raw logs and unpacked data for OperatorSet events raised by the Accountant contract.
type AccountantOperatorSetIterator struct {
	Event *AccountantOperatorSet // Event containing the contract specifics and raw log

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
func (it *AccountantOperatorSetIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AccountantOperatorSet)
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
		it.Event = new(AccountantOperatorSet)
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
func (it *AccountantOperatorSetIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AccountantOperatorSetIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AccountantOperatorSet represents a OperatorSet event raised by the Accountant contract.
type AccountantOperatorSet struct {
	Operator common.Address
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterOperatorSet is a free log retrieval operation binding the contract event 0x99d737e0adf2c449d71890b86772885ec7959b152ddb265f76325b6e68e105d3.
//
// Solidity: event OperatorSet(address indexed operator)
func (_Accountant *AccountantFilterer) FilterOperatorSet(opts *bind.FilterOpts, operator []common.Address) (*AccountantOperatorSetIterator, error) {

	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}

	logs, sub, err := _Accountant.contract.FilterLogs(opts, "OperatorSet", operatorRule)
	if err != nil {
		return nil, err
	}
	return &AccountantOperatorSetIterator{contract: _Accountant.contract, event: "OperatorSet", logs: logs, sub: sub}, nil
}

// WatchOperatorSet is a free log subscription operation binding the contract event 0x99d737e0adf2c449d71890b86772885ec7959b152ddb265f76325b6e68e105d3.
//
// Solidity: event OperatorSet(address indexed operator)
func (_Accountant *AccountantFilterer) WatchOperatorSet(opts *bind.WatchOpts, sink chan<- *AccountantOperatorSet, operator []common.Address) (event.Subscription, error) {

	var operatorRule []interface{}
	for _, operatorItem := range operator {
		operatorRule = append(operatorRule, operatorItem)
	}

	logs, sub, err := _Accountant.contract.WatchLogs(opts, "OperatorSet", operatorRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AccountantOperatorSet)
				if err := _Accountant.contract.UnpackLog(event, "OperatorSet", log); err != nil {
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

// ParseOperatorSet is a log parse operation binding the contract event 0x99d737e0adf2c449d71890b86772885ec7959b152ddb265f76325b6e68e105d3.
//
// Solidity: event OperatorSet(address indexed operator)
func (_Accountant *AccountantFilterer) ParseOperatorSet(log types.Log) (*AccountantOperatorSet, error) {
	event := new(AccountantOperatorSet)
	if err := _Accountant.contract.UnpackLog(event, "OperatorSet", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AccountantOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the Accountant contract.
type AccountantOwnershipTransferredIterator struct {
	Event *AccountantOwnershipTransferred // Event containing the contract specifics and raw log

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
func (it *AccountantOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AccountantOwnershipTransferred)
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
		it.Event = new(AccountantOwnershipTransferred)
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
func (it *AccountantOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AccountantOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AccountantOwnershipTransferred represents a OwnershipTransferred event raised by the Accountant contract.
type AccountantOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Accountant *AccountantFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*AccountantOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Accountant.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &AccountantOwnershipTransferredIterator{contract: _Accountant.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_Accountant *AccountantFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *AccountantOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _Accountant.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AccountantOwnershipTransferred)
				if err := _Accountant.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
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
func (_Accountant *AccountantFilterer) ParseOwnershipTransferred(log types.Log) (*AccountantOwnershipTransferred, error) {
	event := new(AccountantOwnershipTransferred)
	if err := _Accountant.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AccountantRegistrationIterator is returned from FilterRegistration and is used to iterate over the raw logs and unpacked data for Registration events raised by the Accountant contract.
type AccountantRegistrationIterator struct {
	Event *AccountantRegistration // Event containing the contract specifics and raw log

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
func (it *AccountantRegistrationIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AccountantRegistration)
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
		it.Event = new(AccountantRegistration)
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
func (it *AccountantRegistrationIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AccountantRegistrationIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AccountantRegistration represents a Registration event raised by the Accountant contract.
type AccountantRegistration struct {
	User     common.Address
	ExpireAt *big.Int
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterRegistration is a free log retrieval operation binding the contract event 0xf6c60b059f60c86b2d612b237ac38bd87bc1f68f87600cd360d68351af1ca95f.
//
// Solidity: event Registration(address indexed user, uint256 expireAt)
func (_Accountant *AccountantFilterer) FilterRegistration(opts *bind.FilterOpts, user []common.Address) (*AccountantRegistrationIterator, error) {

	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	logs, sub, err := _Accountant.contract.FilterLogs(opts, "Registration", userRule)
	if err != nil {
		return nil, err
	}
	return &AccountantRegistrationIterator{contract: _Accountant.contract, event: "Registration", logs: logs, sub: sub}, nil
}

// WatchRegistration is a free log subscription operation binding the contract event 0xf6c60b059f60c86b2d612b237ac38bd87bc1f68f87600cd360d68351af1ca95f.
//
// Solidity: event Registration(address indexed user, uint256 expireAt)
func (_Accountant *AccountantFilterer) WatchRegistration(opts *bind.WatchOpts, sink chan<- *AccountantRegistration, user []common.Address) (event.Subscription, error) {

	var userRule []interface{}
	for _, userItem := range user {
		userRule = append(userRule, userItem)
	}

	logs, sub, err := _Accountant.contract.WatchLogs(opts, "Registration", userRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AccountantRegistration)
				if err := _Accountant.contract.UnpackLog(event, "Registration", log); err != nil {
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

// ParseRegistration is a log parse operation binding the contract event 0xf6c60b059f60c86b2d612b237ac38bd87bc1f68f87600cd360d68351af1ca95f.
//
// Solidity: event Registration(address indexed user, uint256 expireAt)
func (_Accountant *AccountantFilterer) ParseRegistration(log types.Log) (*AccountantRegistration, error) {
	event := new(AccountantRegistration)
	if err := _Accountant.contract.UnpackLog(event, "Registration", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AccountantTrustedClientAddedIterator is returned from FilterTrustedClientAdded and is used to iterate over the raw logs and unpacked data for TrustedClientAdded events raised by the Accountant contract.
type AccountantTrustedClientAddedIterator struct {
	Event *AccountantTrustedClientAdded // Event containing the contract specifics and raw log

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
func (it *AccountantTrustedClientAddedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AccountantTrustedClientAdded)
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
		it.Event = new(AccountantTrustedClientAdded)
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
func (it *AccountantTrustedClientAddedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AccountantTrustedClientAddedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AccountantTrustedClientAdded represents a TrustedClientAdded event raised by the Accountant contract.
type AccountantTrustedClientAdded struct {
	Client common.Address
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterTrustedClientAdded is a free log retrieval operation binding the contract event 0x231c2c4d122e90f967a7b0561eca88c75428b96b2d2eb6a4eb5cc76693beba5a.
//
// Solidity: event TrustedClientAdded(address indexed client)
func (_Accountant *AccountantFilterer) FilterTrustedClientAdded(opts *bind.FilterOpts, client []common.Address) (*AccountantTrustedClientAddedIterator, error) {

	var clientRule []interface{}
	for _, clientItem := range client {
		clientRule = append(clientRule, clientItem)
	}

	logs, sub, err := _Accountant.contract.FilterLogs(opts, "TrustedClientAdded", clientRule)
	if err != nil {
		return nil, err
	}
	return &AccountantTrustedClientAddedIterator{contract: _Accountant.contract, event: "TrustedClientAdded", logs: logs, sub: sub}, nil
}

// WatchTrustedClientAdded is a free log subscription operation binding the contract event 0x231c2c4d122e90f967a7b0561eca88c75428b96b2d2eb6a4eb5cc76693beba5a.
//
// Solidity: event TrustedClientAdded(address indexed client)
func (_Accountant *AccountantFilterer) WatchTrustedClientAdded(opts *bind.WatchOpts, sink chan<- *AccountantTrustedClientAdded, client []common.Address) (event.Subscription, error) {

	var clientRule []interface{}
	for _, clientItem := range client {
		clientRule = append(clientRule, clientItem)
	}

	logs, sub, err := _Accountant.contract.WatchLogs(opts, "TrustedClientAdded", clientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AccountantTrustedClientAdded)
				if err := _Accountant.contract.UnpackLog(event, "TrustedClientAdded", log); err != nil {
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

// ParseTrustedClientAdded is a log parse operation binding the contract event 0x231c2c4d122e90f967a7b0561eca88c75428b96b2d2eb6a4eb5cc76693beba5a.
//
// Solidity: event TrustedClientAdded(address indexed client)
func (_Accountant *AccountantFilterer) ParseTrustedClientAdded(log types.Log) (*AccountantTrustedClientAdded, error) {
	event := new(AccountantTrustedClientAdded)
	if err := _Accountant.contract.UnpackLog(event, "TrustedClientAdded", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AccountantTrustedClientRemovedIterator is returned from FilterTrustedClientRemoved and is used to iterate over the raw logs and unpacked data for TrustedClientRemoved events raised by the Accountant contract.
type AccountantTrustedClientRemovedIterator struct {
	Event *AccountantTrustedClientRemoved // Event containing the contract specifics and raw log

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
func (it *AccountantTrustedClientRemovedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AccountantTrustedClientRemoved)
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
		it.Event = new(AccountantTrustedClientRemoved)
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
func (it *AccountantTrustedClientRemovedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AccountantTrustedClientRemovedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AccountantTrustedClientRemoved represents a TrustedClientRemoved event raised by the Accountant contract.
type AccountantTrustedClientRemoved struct {
	Client common.Address
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterTrustedClientRemoved is a free log retrieval operation binding the contract event 0x7312fd775ac41b9250b16f48d8e635241805eefa84b4618e737118609464fe94.
//
// Solidity: event TrustedClientRemoved(address indexed client)
func (_Accountant *AccountantFilterer) FilterTrustedClientRemoved(opts *bind.FilterOpts, client []common.Address) (*AccountantTrustedClientRemovedIterator, error) {

	var clientRule []interface{}
	for _, clientItem := range client {
		clientRule = append(clientRule, clientItem)
	}

	logs, sub, err := _Accountant.contract.FilterLogs(opts, "TrustedClientRemoved", clientRule)
	if err != nil {
		return nil, err
	}
	return &AccountantTrustedClientRemovedIterator{contract: _Accountant.contract, event: "TrustedClientRemoved", logs: logs, sub: sub}, nil
}

// WatchTrustedClientRemoved is a free log subscription operation binding the contract event 0x7312fd775ac41b9250b16f48d8e635241805eefa84b4618e737118609464fe94.
//
// Solidity: event TrustedClientRemoved(address indexed client)
func (_Accountant *AccountantFilterer) WatchTrustedClientRemoved(opts *bind.WatchOpts, sink chan<- *AccountantTrustedClientRemoved, client []common.Address) (event.Subscription, error) {

	var clientRule []interface{}
	for _, clientItem := range client {
		clientRule = append(clientRule, clientItem)
	}

	logs, sub, err := _Accountant.contract.WatchLogs(opts, "TrustedClientRemoved", clientRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AccountantTrustedClientRemoved)
				if err := _Accountant.contract.UnpackLog(event, "TrustedClientRemoved", log); err != nil {
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

// ParseTrustedClientRemoved is a log parse operation binding the contract event 0x7312fd775ac41b9250b16f48d8e635241805eefa84b4618e737118609464fe94.
//
// Solidity: event TrustedClientRemoved(address indexed client)
func (_Accountant *AccountantFilterer) ParseTrustedClientRemoved(log types.Log) (*AccountantTrustedClientRemoved, error) {
	event := new(AccountantTrustedClientRemoved)
	if err := _Accountant.contract.UnpackLog(event, "TrustedClientRemoved", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// AccountantValidPeriodUpdatedIterator is returned from FilterValidPeriodUpdated and is used to iterate over the raw logs and unpacked data for ValidPeriodUpdated events raised by the Accountant contract.
type AccountantValidPeriodUpdatedIterator struct {
	Event *AccountantValidPeriodUpdated // Event containing the contract specifics and raw log

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
func (it *AccountantValidPeriodUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(AccountantValidPeriodUpdated)
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
		it.Event = new(AccountantValidPeriodUpdated)
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
func (it *AccountantValidPeriodUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *AccountantValidPeriodUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// AccountantValidPeriodUpdated represents a ValidPeriodUpdated event raised by the Accountant contract.
type AccountantValidPeriodUpdated struct {
	Period *big.Int
	Raw    types.Log // Blockchain specific contextual infos
}

// FilterValidPeriodUpdated is a free log retrieval operation binding the contract event 0x212484687070ba3010539da0b1227e55b6cd6be5b7561d8aedb1afc1c04d1abb.
//
// Solidity: event ValidPeriodUpdated(uint256 period)
func (_Accountant *AccountantFilterer) FilterValidPeriodUpdated(opts *bind.FilterOpts) (*AccountantValidPeriodUpdatedIterator, error) {

	logs, sub, err := _Accountant.contract.FilterLogs(opts, "ValidPeriodUpdated")
	if err != nil {
		return nil, err
	}
	return &AccountantValidPeriodUpdatedIterator{contract: _Accountant.contract, event: "ValidPeriodUpdated", logs: logs, sub: sub}, nil
}

// WatchValidPeriodUpdated is a free log subscription operation binding the contract event 0x212484687070ba3010539da0b1227e55b6cd6be5b7561d8aedb1afc1c04d1abb.
//
// Solidity: event ValidPeriodUpdated(uint256 period)
func (_Accountant *AccountantFilterer) WatchValidPeriodUpdated(opts *bind.WatchOpts, sink chan<- *AccountantValidPeriodUpdated) (event.Subscription, error) {

	logs, sub, err := _Accountant.contract.WatchLogs(opts, "ValidPeriodUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(AccountantValidPeriodUpdated)
				if err := _Accountant.contract.UnpackLog(event, "ValidPeriodUpdated", log); err != nil {
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

// ParseValidPeriodUpdated is a log parse operation binding the contract event 0x212484687070ba3010539da0b1227e55b6cd6be5b7561d8aedb1afc1c04d1abb.
//
// Solidity: event ValidPeriodUpdated(uint256 period)
func (_Accountant *AccountantFilterer) ParseValidPeriodUpdated(log types.Log) (*AccountantValidPeriodUpdated, error) {
	event := new(AccountantValidPeriodUpdated)
	if err := _Accountant.contract.UnpackLog(event, "ValidPeriodUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
