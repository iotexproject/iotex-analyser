// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package main

import (
	"errors"
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
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
)

// DelegateProfileMetaData contains all meta data concerning the DelegateProfile contract.
var DelegateProfileMetaData = &bind.MetaData{
	ABI: "[{\"constant\":false,\"inputs\":[{\"name\":\"_delegate\",\"type\":\"address\"},{\"name\":\"_name\",\"type\":\"string\"},{\"name\":\"_value\",\"type\":\"bytes\"}],\"name\":\"updateProfileForDelegate\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"register\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_delegate\",\"type\":\"address\"}],\"name\":\"getEncodedProfile\",\"outputs\":[{\"name\":\"code_\",\"type\":\"bytes\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_address\",\"type\":\"address\"}],\"name\":\"isOwner\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_name\",\"type\":\"string\"}],\"name\":\"getFieldByName\",\"outputs\":[{\"name\":\"verifier_\",\"type\":\"address\"},{\"name\":\"deprecated_\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_byteCode\",\"type\":\"bytes\"}],\"name\":\"updateProfileWithByteCode\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"withdraw\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"unpause\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_name\",\"type\":\"string\"},{\"name\":\"_verifierAddr\",\"type\":\"address\"}],\"name\":\"newField\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_name\",\"type\":\"string\"},{\"name\":\"_value\",\"type\":\"bytes\"}],\"name\":\"updateProfile\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"pause\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_idx\",\"type\":\"uint256\"}],\"name\":\"getFieldByIndex\",\"outputs\":[{\"name\":\"name_\",\"type\":\"string\"},{\"name\":\"verifier_\",\"type\":\"address\"},{\"name\":\"deprecated_\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_delegate\",\"type\":\"address\"},{\"name\":\"_byteCode\",\"type\":\"bytes\"}],\"name\":\"updateProfileWithByteCodeForDelegate\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"fieldNames\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[{\"name\":\"_delegate\",\"type\":\"address\"},{\"name\":\"_field\",\"type\":\"string\"}],\"name\":\"getProfileByField\",\"outputs\":[{\"name\":\"\",\"type\":\"bytes\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_name\",\"type\":\"string\"}],\"name\":\"deprecateField\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"numOfFields\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"name\":\"registerAddr\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"fee\",\"type\":\"uint256\"}],\"name\":\"FeeUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"delegate\",\"type\":\"address\"},{\"indexed\":false,\"name\":\"name\",\"type\":\"string\"},{\"indexed\":false,\"name\":\"value\",\"type\":\"bytes\"}],\"name\":\"ProfileUpdated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"name\",\"type\":\"string\"}],\"name\":\"FieldDeprecated\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"name\":\"name\",\"type\":\"string\"}],\"name\":\"NewField\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[],\"name\":\"Pause\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[],\"name\":\"Unpause\",\"type\":\"event\"}]",
	Sigs: map[string]string{
		"e0adf839": "deprecateField(string)",
		"b2d3dd66": "fieldNames(uint256)",
		"2652877e": "getEncodedProfile(address)",
		"8ac834a3": "getFieldByIndex(uint256)",
		"363d62dd": "getFieldByName(string)",
		"cdcb1d52": "getProfileByField(address,string)",
		"2f54bf6e": "isOwner(address)",
		"68beafc8": "newField(string,address)",
		"e6ce112f": "numOfFields()",
		"8da5cb5b": "owner()",
		"8456cb59": "pause()",
		"5c975abb": "paused()",
		"1aa3a008": "register()",
		"f2fde38b": "transferOwnership(address)",
		"3f4ba83a": "unpause()",
		"6eeb9b10": "updateProfile(string,bytes)",
		"199baa71": "updateProfileForDelegate(address,string,bytes)",
		"37d1f437": "updateProfileWithByteCode(bytes)",
		"ac468ebc": "updateProfileWithByteCodeForDelegate(address,bytes)",
		"3ccfd60b": "withdraw()",
	},
	Bin: "0x60806040526000805460a060020a60ff021916905534801561002057600080fd5b506040516020806122bb833981016040525160008054600160a060020a0319908116331790915560018054600160a060020a039093169290911691909117905561224c8061006f6000396000f3006080604052600436106101115763ffffffff7c0100000000000000000000000000000000000000000000000000000000600035041663199baa7181146101165780631aa3a008146101bd5780632652877e146101ee5780632f54bf6e14610284578063363d62dd146102b957806337d1f437146103355780633ccfd60b1461038e5780633f4ba83a146103a35780635c975abb146103b857806368beafc8146103cd5780636eeb9b10146104315780638456cb59146104c85780638ac834a3146104dd5780638da5cb5b14610591578063ac468ebc146105a6578063b2d3dd661461060d578063cdcb1d5214610625578063e0adf8391461068c578063e6ce112f146106e5578063f2fde38b1461070c575b600080fd5b34801561012257600080fd5b5060408051602060046024803582810135601f81018590048502860185019096528585526101bb958335600160a060020a031695369560449491939091019190819084018382808284375050604080516020601f89358b018035918201839004830284018301909452808352979a99988101979196509182019450925082915084018382808284375094975061072d9650505050505050565b005b3480156101c957600080fd5b506101d261079e565b60408051600160a060020a039092168252519081900360200190f35b3480156101fa57600080fd5b5061020f600160a060020a03600435166107ad565b6040805160208082528351818301528351919283929083019185019080838360005b83811015610249578181015183820152602001610231565b50505050905090810190601f1680156102765780820380516001836020036101000a031916815260200191505b509250505060405180910390f35b34801561029057600080fd5b506102a5600160a060020a0360043516610bac565b604080519115158252519081900360200190f35b3480156102c557600080fd5b506040805160206004803580820135601f8101849004840285018401909552848452610312943694929360249392840191908190840183828082843750949750610bc39650505050505050565b60408051600160a060020a03909316835290151560208301528051918290030190f35b34801561034157600080fd5b506040805160206004803580820135601f81018490048402850184019095528484526101bb943694929360249392840191908190840183828082843750949750610d659650505050505050565b34801561039a57600080fd5b506101bb610dd6565b3480156103af57600080fd5b506101bb610e17565b3480156103c457600080fd5b506102a5610e8a565b3480156103d957600080fd5b506040805160206004803580820135601f81018490048402850184019095528484526101bb94369492936024939284019190819084018382808284375094975050509235600160a060020a03169350610e9a92505050565b34801561043d57600080fd5b506040805160206004803580820135601f81018490048402850184019095528484526101bb94369492936024939284019190819084018382808284375050604080516020601f89358b018035918201839004830284018301909452808352979a999881019791965091820194509250829150840183828082843750949750610f179650505050505050565b3480156104d457600080fd5b506101bb610f86565b3480156104e957600080fd5b506104f5600435610ffe565b604051808060200184600160a060020a0316600160a060020a0316815260200183151515158152602001828103825285818151815260200191508051906020019080838360005b8381101561055457818101518382015260200161053c565b50505050905090810190601f1680156105815780820380516001836020036101000a031916815260200191505b5094505050505060405180910390f35b34801561059d57600080fd5b506101d26111e3565b3480156105b257600080fd5b5060408051602060046024803582810135601f81018590048502860185019096528585526101bb958335600160a060020a03169536956044949193909101919081908401838280828437509497506111f29650505050505050565b34801561061957600080fd5b5061020f600435611210565b34801561063157600080fd5b5060408051602060046024803582810135601f810185900485028601850190965285855261020f958335600160a060020a03169536956044949193909101919081908401838280828437509497506112b79650505050505050565b34801561069857600080fd5b506040805160206004803580820135601f81018490048402850184019095528484526101bb94369492936024939284019190819084018382808284375094975061141c9650505050505050565b3480156106f157600080fd5b506106fa611614565b60408051918252519081900360200190f35b34801561071857600080fd5b506101bb600160a060020a036004351661161b565b61073633610bac565b151561074157600080fd5b61074a8361165e565b151561078e576040805160e560020a62461bcd02815260206004820152600e6024820152600080516020612201833981519152604482015290519081900360640190fd5b610799838383611719565b505050565b600154600160a060020a031681565b6060600080606060006107bf8661165e565b1515610803576040805160e560020a62461bcd02815260206004820152600e6024820152600080516020612201833981519152604482015290519081900360640190fd5b60009350600092505b60035483101561097b57600380548490811061082457fe5b600091825260209182902001805460408051601f60026000196101006001871615020190941693909304928301859004850281018501909152818152928301828280156108b25780601f10610887576101008083540402835291602001916108b2565b820191906000526020600020905b81548152906001019060200180831161089557829003601f168201915b505050505091506108c282611b2a565b1515610970576002600087600160a060020a0316600160a060020a03168152602001908152602001600020826040518082805190602001908083835b6020831061091d5780518252601f1990920191602091820191016108fe565b6001836020036101000a0380198251168184511680821785525050505050509050019150509081526020016040518091039020805460018160011615610100020316600290049050825160400101840193505b60019092019161080c565b836040519080825280601f01601f1916602001820160405280156109a9578160200160208202803883390190505b5094506000925050602084015b600354831015610ba35760038054849081106109ce57fe5b600091825260209182902001805460408051601f6002600019610100600187161502019094169390930492830185900485028101850190915281815292830182828015610a5c5780601f10610a3157610100808354040283529160200191610a5c565b820191906000526020600020905b815481529060010190602001808311610a3f57829003601f168201915b50505050509150610a6c82611b2a565b1515610b9857610a7c8183611c5e565b9050610b95816002600089600160a060020a0316600160a060020a03168152602001908152602001600020846040518082805190602001908083835b60208310610ad75780518252601f199092019160209182019101610ab8565b518151600019602094850361010090810a820192831692199390931691909117909252949092019687526040805197889003820188208054601f6002600183161590980290950116959095049283018290048202880182019052818752929450925050830182828015610b8b5780601f10610b6057610100808354040283529160200191610b8b565b820191906000526020600020905b815481529060010190602001808311610b6e57829003601f168201915b5050505050611c5e565b90505b6001909201916109b6565b50505050919050565b600054600160a060020a038281169116145b919050565b6000806004836040518082805190602001908083835b60208310610bf85780518252601f199092019160209182019101610bd9565b51815160209384036101000a600019018019909216911617905292019485525060405193849003019092205460a860020a900460ff1615159150610c889050576040805160e560020a62461bcd02815260206004820152601460248201527f756e646566696e6564206669656c64206e616d65000000000000000000000000604482015290519081900360640190fd5b6004836040518082805190602001908083835b60208310610cba5780518252601f199092019160209182019101610c9b565b51815160209384036101000a60001901801990921691161790529201948552506040519384900381018420548751600160a060020a039091169650600494889450925082918401908083835b60208310610d255780518252601f199092019160209182019101610d06565b51815160209384036101000a6000190180199092169116179052920194855250604051938490030190922054939560a060020a90940460ff169450505050565b610d6e3361165e565b1515610db2576040805160e560020a62461bcd02815260206004820152600e6024820152600080516020612201833981519152604482015290519081900360640190fd5b60005460a060020a900460ff1615610dc957600080fd5b610dd33382611cbe565b50565b610ddf33610bac565b1515610dea57600080fd5b6040513390303180156108fc02916000818181858888f19350505050158015610dd3573d6000803e3d6000fd5b610e2033610bac565b1515610e2b57600080fd5b60005460a060020a900460ff161515610e4357600080fd5b6000805474ff0000000000000000000000000000000000000000191681556040517f7805862f689e2f13df9f062ff482ad3ad112aca9e0847911ed832e158c525b339190a1565b60005460a060020a900460ff1681565b610ea333610bac565b1515610eae57600080fd5b8151600010610f07576040805160e560020a62461bcd02815260206004820152601a60248201527f6669656c64206e616d652063616e6e6f7420626520656d707479000000000000604482015290519081900360640190fd5b610f1382826000611d7a565b5050565b610f203361165e565b1515610f64576040805160e560020a62461bcd02815260206004820152600e6024820152600080516020612201833981519152604482015290519081900360640190fd5b60005460a060020a900460ff1615610f7b57600080fd5b610f13338383611719565b610f8f33610bac565b1515610f9a57600080fd5b60005460a060020a900460ff1615610fb157600080fd5b6000805474ff0000000000000000000000000000000000000000191660a060020a1781556040517f6985a02210a168e66602d3235cb6db0e70f92b3ba4d376a33c0f3d9434bff6259190a1565b606060008061100b611614565b8410611061576040805160e560020a62461bcd02815260206004820152601b60248201527f6669656c6420696e646578206f7574206f6620626f756e646172790000000000604482015290519081900360640190fd5b600380548590811061106f57fe5b600091825260209182902001805460408051601f60026000196101006001871615020190941693909304928301859004850281018501909152818152928301828280156110fd5780601f106110d2576101008083540402835291602001916110fd565b820191906000526020600020905b8154815290600101906020018083116110e057829003601f168201915b505050505092506004836040518082805190602001908083835b602083106111365780518252601f199092019160209182019101611117565b51815160209384036101000a60001901801990921691161790529201948552506040519384900381018420548751600160a060020a039091169650600494889450925082918401908083835b602083106111a15780518252601f199092019160209182019101611182565b51815160209384036101000a60001901801990921691161790529201948552506040519384900301909220549496939550505060a060020a90920460ff169150565b600054600160a060020a031681565b6111fb33610bac565b151561120657600080fd5b610f138282611cbe565b600380548290811061121e57fe5b600091825260209182902001805460408051601f60026000196101006001871615020190941693909304928301859004850281018501909152818152935090918301828280156112af5780601f10611284576101008083540402835291602001916112af565b820191906000526020600020905b81548152906001019060200180831161129257829003601f168201915b505050505081565b60606112c28361165e565b1515611306576040805160e560020a62461bcd02815260206004820152600e6024820152600080516020612201833981519152604482015290519081900360640190fd5b6002600084600160a060020a0316600160a060020a03168152602001908152602001600020826040518082805190602001908083835b6020831061135b5780518252601f19909201916020918201910161133c565b518151600019602094850361010090810a820192831692199390931691909117909252949092019687526040805197889003820188208054601f600260018316159098029095011695909504928301829004820288018201905281875292945092505083018282801561140f5780601f106113e45761010080835404028352916020019161140f565b820191906000526020600020905b8154815290600101906020018083116113f257829003601f168201915b5050505050905092915050565b61142533610bac565b151561143057600080fd5b6004816040518082805190602001908083835b602083106114625780518252601f199092019160209182019101611443565b51815160209384036101000a600019018019909216911617905292019485525060405193849003019092205460a860020a900460ff16151591506114f29050576040805160e560020a62461bcd02815260206004820152600f60248201527f756e646566696e6564206669656c640000000000000000000000000000000000604482015290519081900360640190fd5b60016004826040518082805190602001908083835b602083106115265780518252601f199092019160209182019101611507565b51815160209384036101000a6000190180199092169116179052920194855250604080519485900382018520805474ff0000000000000000000000000000000000000000191660a060020a971515979097029690961790955580845285518482015285517ff47b35d35c737e18368ebfa5496bc97dabcea3e7b0075269da84fc32d0f201b8958795945084935083019185019080838360005b838110156115d75781810151838201526020016115bf565b50505050905090810190601f1680156116045780820380516001836020036101000a031916815260200191505b509250505060405180910390a150565b6003545b90565b61162433610bac565b151561162f57600080fd5b6000805473ffffffffffffffffffffffffffffffffffffffff1916600160a060020a0392909216919091179055565b600154600090600160a060020a0316151561167b57506001610bbe565b600154604080517f5f609695000000000000000000000000000000000000000000000000000000008152600160a060020a03858116600483015291516000939290921691635f6096959160248082019260209290919082900301818787803b1580156116e657600080fd5b505af11580156116fa573d6000803e3d6000fd5b505050506040513d602081101561171057600080fd5b50511192915050565b80516201000011611774576040805160e560020a62461bcd02815260206004820152600e60248201527f76616c756520746f6f206c6f6e67000000000000000000000000000000000000604482015290519081900360640190fd5b61177d82611b2a565b156117d2576040805160e560020a62461bcd02815260206004820152601060248201527f64657072656361746564206669656c6400000000000000000000000000000000604482015290519081900360640190fd5b6004826040518082805190602001908083835b602083106118045780518252601f1990920191602091820191016117e5565b51815160209384036101000a60001901801990921691161790529201948552506040519384900381018420547f8e760afe00000000000000000000000000000000000000000000000000000000855260048501828152865160248701528651600160a060020a0390921695638e760afe9550879450909283926044909201919085019080838360005b838110156118a557818101518382015260200161188d565b50505050905090810190601f1680156118d25780820380516001836020036101000a031916815260200191505b5092505050602060405180830381600087803b1580156118f157600080fd5b505af1158015611905573d6000803e3d6000fd5b505050506040513d602081101561191b57600080fd5b50511515611973576040805160e560020a62461bcd02815260206004820152600d60248201527f696e76616c69642076616c756500000000000000000000000000000000000000604482015290519081900360640190fd5b806002600085600160a060020a0316600160a060020a03168152602001908152602001600020836040518082805190602001908083835b602083106119c95780518252601f1990920191602091820191016119aa565b51815160209384036101000a60001901801990921691161790529201948552506040519384900381019093208451611a0a9591949190910192509050612168565b507f217aa5ef0b78f028d51fd573433bdbe2daf6f8505e6a71f3af1393c8440b341b8383836040518084600160a060020a0316600160a060020a031681526020018060200180602001838103835285818151815260200191508051906020019080838360005b83811015611a88578181015183820152602001611a70565b50505050905090810190601f168015611ab55780820380516001836020036101000a031916815260200191505b50838103825284518152845160209182019186019080838360005b83811015611ae8578181015183820152602001611ad0565b50505050905090810190601f168015611b155780820380516001836020036101000a031916815260200191505b509550505050505060405180910390a1505050565b60006004826040518082805190602001908083835b60208310611b5e5780518252601f199092019160209182019101611b3f565b51815160209384036101000a600019018019909216911617905292019485525060405193849003019092205460a860020a900460ff1615159150611bee9050576040805160e560020a62461bcd02815260206004820152601460248201527f756e646566696e6564206669656c64206e616d65000000000000000000000000604482015290519081900360640190fd5b6004826040518082805190602001908083835b60208310611c205780518252601f199092019160209182019101611c01565b51815160209384036101000a600019018019909216911617905292019485525060405193849003019092205460a060020a900460ff16949350505050565b805160009060200181805b602082018310611c86578185015182870152602082019150611c69565b6001602084066020036101000a0390508019828601511681838801511681811784890152505084518660200101935050505092915050565b6000606080825b8451811015611d7257611cd88582612033565b935060008411611d32576040805160e560020a62461bcd02815260206004820152600e60248201527f696e76616c6964206c656e677468000000000000000000000000000000000000604482015290519081900360640190fd5b602001611d4085828661209c565b92508301611d4e8582612033565b9350602001611d5e85828661209c565b91508301611d6d868484611719565b611cc5565b505050505050565b6004836040518082805190602001908083835b60208310611dac5780518252601f199092019160209182019101611d8d565b51815160209384036101000a600019018019909216911617905292019485525060405193849003019092205460a860020a900460ff16159150611e3b9050576040805160e560020a62461bcd02815260206004820152601460248201527f6475706c6963617465206669656c64206e616d65000000000000000000000000604482015290519081900360640190fd5b60606040519081016040528083600160a060020a031681526020018215158152602001600115158152506004846040518082805190602001908083835b60208310611e975780518252601f199092019160209182019101611e78565b51815160209384036101000a6000190180199092169116179052920194855250604080519485900382019094208551815487840151979096015173ffffffffffffffffffffffffffffffffffffffff19909616600160a060020a039091161774ff0000000000000000000000000000000000000000191660a060020a961515969096029590951775ff000000000000000000000000000000000000000000191660a860020a94151594909402939093179093555060038054600181018083556000929092528651919350611f93927fc2575a0e9e593c00f959f8c92f12db2869c3395a3b0502d05e2516446f71f85b9091019190870190612168565b50507f53096991d49a1876b3be4d7f3d107f7f92043e0fceec1e81b5ba38841d78123b836040518080602001828103825283818151815260200191508051906020019080838360005b83811015611ff4578181015183820152602001611fdc565b50505050905090810190601f1680156120215780820380516001836020036101000a031916815260200191505b509250505060405180910390a1505050565b600081602001835110151515612093576040805160e560020a62461bcd02815260206004820152601160248201527f696e76616c6964206279746520636f6465000000000000000000000000000000604482015290519081900360640190fd5b50016020015190565b6060808284018551101515156120fc576040805160e560020a62461bcd02815260206004820152601160248201527f696e76616c6964206279746520636f6465000000000000000000000000000000604482015290519081900360640190fd5b821580156121155760405191506020820160405261215f565b6040519150601f8416801560200281840101858101878315602002848b0101015b8183101561214e578051835260209283019201612136565b5050858452601f01601f1916604052505b50949350505050565b828054600181600116156101000203166002900490600052602060002090601f016020900481019282601f106121a957805160ff19168380011785556121d6565b828001600101855582156121d6579182015b828111156121d65782518255916020019190600101906121bb565b506121e29291506121e6565b5090565b61161891905b808211156121e257600081556001016121ec56006e6f742072656769737465726564000000000000000000000000000000000000a165627a7a723058205b3606ac6edacfb62cec5e52bca75a3971588097c2c7c904e61608d3bdd7c6800029",
}

// DelegateProfileABI is the input ABI used to generate the binding from.
// Deprecated: Use DelegateProfileMetaData.ABI instead.
var DelegateProfileABI = DelegateProfileMetaData.ABI

// Deprecated: Use DelegateProfileMetaData.Sigs instead.
// DelegateProfileFuncSigs maps the 4-byte function signature to its string representation.
var DelegateProfileFuncSigs = DelegateProfileMetaData.Sigs

// DelegateProfileBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use DelegateProfileMetaData.Bin instead.
var DelegateProfileBin = DelegateProfileMetaData.Bin

// DeployDelegateProfile deploys a new Ethereum contract, binding an instance of DelegateProfile to it.
func DeployDelegateProfile(auth *bind.TransactOpts, backend bind.ContractBackend, registerAddr common.Address) (common.Address, *types.Transaction, *DelegateProfile, error) {
	parsed, err := DelegateProfileMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(DelegateProfileBin), backend, registerAddr)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &DelegateProfile{DelegateProfileCaller: DelegateProfileCaller{contract: contract}, DelegateProfileTransactor: DelegateProfileTransactor{contract: contract}, DelegateProfileFilterer: DelegateProfileFilterer{contract: contract}}, nil
}

// DelegateProfile is an auto generated Go binding around an Ethereum contract.
type DelegateProfile struct {
	DelegateProfileCaller     // Read-only binding to the contract
	DelegateProfileTransactor // Write-only binding to the contract
	DelegateProfileFilterer   // Log filterer for contract events
}

// DelegateProfileCaller is an auto generated read-only Go binding around an Ethereum contract.
type DelegateProfileCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DelegateProfileTransactor is an auto generated write-only Go binding around an Ethereum contract.
type DelegateProfileTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DelegateProfileFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type DelegateProfileFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// DelegateProfileSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type DelegateProfileSession struct {
	Contract     *DelegateProfile  // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// DelegateProfileCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type DelegateProfileCallerSession struct {
	Contract *DelegateProfileCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts          // Call options to use throughout this session
}

// DelegateProfileTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type DelegateProfileTransactorSession struct {
	Contract     *DelegateProfileTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts          // Transaction auth options to use throughout this session
}

// DelegateProfileRaw is an auto generated low-level Go binding around an Ethereum contract.
type DelegateProfileRaw struct {
	Contract *DelegateProfile // Generic contract binding to access the raw methods on
}

// DelegateProfileCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type DelegateProfileCallerRaw struct {
	Contract *DelegateProfileCaller // Generic read-only contract binding to access the raw methods on
}

// DelegateProfileTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type DelegateProfileTransactorRaw struct {
	Contract *DelegateProfileTransactor // Generic write-only contract binding to access the raw methods on
}

// NewDelegateProfile creates a new instance of DelegateProfile, bound to a specific deployed contract.
func NewDelegateProfile(address common.Address, backend bind.ContractBackend) (*DelegateProfile, error) {
	contract, err := bindDelegateProfile(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &DelegateProfile{DelegateProfileCaller: DelegateProfileCaller{contract: contract}, DelegateProfileTransactor: DelegateProfileTransactor{contract: contract}, DelegateProfileFilterer: DelegateProfileFilterer{contract: contract}}, nil
}

// NewDelegateProfileCaller creates a new read-only instance of DelegateProfile, bound to a specific deployed contract.
func NewDelegateProfileCaller(address common.Address, caller bind.ContractCaller) (*DelegateProfileCaller, error) {
	contract, err := bindDelegateProfile(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DelegateProfileCaller{contract: contract}, nil
}

// NewDelegateProfileTransactor creates a new write-only instance of DelegateProfile, bound to a specific deployed contract.
func NewDelegateProfileTransactor(address common.Address, transactor bind.ContractTransactor) (*DelegateProfileTransactor, error) {
	contract, err := bindDelegateProfile(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &DelegateProfileTransactor{contract: contract}, nil
}

// NewDelegateProfileFilterer creates a new log filterer instance of DelegateProfile, bound to a specific deployed contract.
func NewDelegateProfileFilterer(address common.Address, filterer bind.ContractFilterer) (*DelegateProfileFilterer, error) {
	contract, err := bindDelegateProfile(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &DelegateProfileFilterer{contract: contract}, nil
}

// bindDelegateProfile binds a generic wrapper to an already deployed contract.
func bindDelegateProfile(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(DelegateProfileABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DelegateProfile *DelegateProfileRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DelegateProfile.Contract.DelegateProfileCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DelegateProfile *DelegateProfileRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DelegateProfile.Contract.DelegateProfileTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DelegateProfile *DelegateProfileRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DelegateProfile.Contract.DelegateProfileTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_DelegateProfile *DelegateProfileCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _DelegateProfile.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_DelegateProfile *DelegateProfileTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DelegateProfile.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_DelegateProfile *DelegateProfileTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _DelegateProfile.Contract.contract.Transact(opts, method, params...)
}

// FieldNames is a free data retrieval call binding the contract method 0xb2d3dd66.
//
// Solidity: function fieldNames(uint256 ) view returns(string)
func (_DelegateProfile *DelegateProfileCaller) FieldNames(opts *bind.CallOpts, arg0 *big.Int) (string, error) {
	var out []interface{}
	err := _DelegateProfile.contract.Call(opts, &out, "fieldNames", arg0)

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// FieldNames is a free data retrieval call binding the contract method 0xb2d3dd66.
//
// Solidity: function fieldNames(uint256 ) view returns(string)
func (_DelegateProfile *DelegateProfileSession) FieldNames(arg0 *big.Int) (string, error) {
	return _DelegateProfile.Contract.FieldNames(&_DelegateProfile.CallOpts, arg0)
}

// FieldNames is a free data retrieval call binding the contract method 0xb2d3dd66.
//
// Solidity: function fieldNames(uint256 ) view returns(string)
func (_DelegateProfile *DelegateProfileCallerSession) FieldNames(arg0 *big.Int) (string, error) {
	return _DelegateProfile.Contract.FieldNames(&_DelegateProfile.CallOpts, arg0)
}

// GetEncodedProfile is a free data retrieval call binding the contract method 0x2652877e.
//
// Solidity: function getEncodedProfile(address _delegate) view returns(bytes code_)
func (_DelegateProfile *DelegateProfileCaller) GetEncodedProfile(opts *bind.CallOpts, _delegate common.Address) ([]byte, error) {
	var out []interface{}
	err := _DelegateProfile.contract.Call(opts, &out, "getEncodedProfile", _delegate)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// GetEncodedProfile is a free data retrieval call binding the contract method 0x2652877e.
//
// Solidity: function getEncodedProfile(address _delegate) view returns(bytes code_)
func (_DelegateProfile *DelegateProfileSession) GetEncodedProfile(_delegate common.Address) ([]byte, error) {
	return _DelegateProfile.Contract.GetEncodedProfile(&_DelegateProfile.CallOpts, _delegate)
}

// GetEncodedProfile is a free data retrieval call binding the contract method 0x2652877e.
//
// Solidity: function getEncodedProfile(address _delegate) view returns(bytes code_)
func (_DelegateProfile *DelegateProfileCallerSession) GetEncodedProfile(_delegate common.Address) ([]byte, error) {
	return _DelegateProfile.Contract.GetEncodedProfile(&_DelegateProfile.CallOpts, _delegate)
}

// GetFieldByIndex is a free data retrieval call binding the contract method 0x8ac834a3.
//
// Solidity: function getFieldByIndex(uint256 _idx) view returns(string name_, address verifier_, bool deprecated_)
func (_DelegateProfile *DelegateProfileCaller) GetFieldByIndex(opts *bind.CallOpts, _idx *big.Int) (struct {
	Name       string
	Verifier   common.Address
	Deprecated bool
}, error) {
	var out []interface{}
	err := _DelegateProfile.contract.Call(opts, &out, "getFieldByIndex", _idx)

	outstruct := new(struct {
		Name       string
		Verifier   common.Address
		Deprecated bool
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Name = *abi.ConvertType(out[0], new(string)).(*string)
	outstruct.Verifier = *abi.ConvertType(out[1], new(common.Address)).(*common.Address)
	outstruct.Deprecated = *abi.ConvertType(out[2], new(bool)).(*bool)

	return *outstruct, err

}

// GetFieldByIndex is a free data retrieval call binding the contract method 0x8ac834a3.
//
// Solidity: function getFieldByIndex(uint256 _idx) view returns(string name_, address verifier_, bool deprecated_)
func (_DelegateProfile *DelegateProfileSession) GetFieldByIndex(_idx *big.Int) (struct {
	Name       string
	Verifier   common.Address
	Deprecated bool
}, error) {
	return _DelegateProfile.Contract.GetFieldByIndex(&_DelegateProfile.CallOpts, _idx)
}

// GetFieldByIndex is a free data retrieval call binding the contract method 0x8ac834a3.
//
// Solidity: function getFieldByIndex(uint256 _idx) view returns(string name_, address verifier_, bool deprecated_)
func (_DelegateProfile *DelegateProfileCallerSession) GetFieldByIndex(_idx *big.Int) (struct {
	Name       string
	Verifier   common.Address
	Deprecated bool
}, error) {
	return _DelegateProfile.Contract.GetFieldByIndex(&_DelegateProfile.CallOpts, _idx)
}

// GetFieldByName is a free data retrieval call binding the contract method 0x363d62dd.
//
// Solidity: function getFieldByName(string _name) view returns(address verifier_, bool deprecated_)
func (_DelegateProfile *DelegateProfileCaller) GetFieldByName(opts *bind.CallOpts, _name string) (struct {
	Verifier   common.Address
	Deprecated bool
}, error) {
	var out []interface{}
	err := _DelegateProfile.contract.Call(opts, &out, "getFieldByName", _name)

	outstruct := new(struct {
		Verifier   common.Address
		Deprecated bool
	})
	if err != nil {
		return *outstruct, err
	}

	outstruct.Verifier = *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	outstruct.Deprecated = *abi.ConvertType(out[1], new(bool)).(*bool)

	return *outstruct, err

}

// GetFieldByName is a free data retrieval call binding the contract method 0x363d62dd.
//
// Solidity: function getFieldByName(string _name) view returns(address verifier_, bool deprecated_)
func (_DelegateProfile *DelegateProfileSession) GetFieldByName(_name string) (struct {
	Verifier   common.Address
	Deprecated bool
}, error) {
	return _DelegateProfile.Contract.GetFieldByName(&_DelegateProfile.CallOpts, _name)
}

// GetFieldByName is a free data retrieval call binding the contract method 0x363d62dd.
//
// Solidity: function getFieldByName(string _name) view returns(address verifier_, bool deprecated_)
func (_DelegateProfile *DelegateProfileCallerSession) GetFieldByName(_name string) (struct {
	Verifier   common.Address
	Deprecated bool
}, error) {
	return _DelegateProfile.Contract.GetFieldByName(&_DelegateProfile.CallOpts, _name)
}

// GetProfileByField is a free data retrieval call binding the contract method 0xcdcb1d52.
//
// Solidity: function getProfileByField(address _delegate, string _field) view returns(bytes)
func (_DelegateProfile *DelegateProfileCaller) GetProfileByField(opts *bind.CallOpts, _delegate common.Address, _field string) ([]byte, error) {
	var out []interface{}
	err := _DelegateProfile.contract.Call(opts, &out, "getProfileByField", _delegate, _field)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// GetProfileByField is a free data retrieval call binding the contract method 0xcdcb1d52.
//
// Solidity: function getProfileByField(address _delegate, string _field) view returns(bytes)
func (_DelegateProfile *DelegateProfileSession) GetProfileByField(_delegate common.Address, _field string) ([]byte, error) {
	return _DelegateProfile.Contract.GetProfileByField(&_DelegateProfile.CallOpts, _delegate, _field)
}

// GetProfileByField is a free data retrieval call binding the contract method 0xcdcb1d52.
//
// Solidity: function getProfileByField(address _delegate, string _field) view returns(bytes)
func (_DelegateProfile *DelegateProfileCallerSession) GetProfileByField(_delegate common.Address, _field string) ([]byte, error) {
	return _DelegateProfile.Contract.GetProfileByField(&_DelegateProfile.CallOpts, _delegate, _field)
}

// IsOwner is a free data retrieval call binding the contract method 0x2f54bf6e.
//
// Solidity: function isOwner(address _address) view returns(bool)
func (_DelegateProfile *DelegateProfileCaller) IsOwner(opts *bind.CallOpts, _address common.Address) (bool, error) {
	var out []interface{}
	err := _DelegateProfile.contract.Call(opts, &out, "isOwner", _address)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsOwner is a free data retrieval call binding the contract method 0x2f54bf6e.
//
// Solidity: function isOwner(address _address) view returns(bool)
func (_DelegateProfile *DelegateProfileSession) IsOwner(_address common.Address) (bool, error) {
	return _DelegateProfile.Contract.IsOwner(&_DelegateProfile.CallOpts, _address)
}

// IsOwner is a free data retrieval call binding the contract method 0x2f54bf6e.
//
// Solidity: function isOwner(address _address) view returns(bool)
func (_DelegateProfile *DelegateProfileCallerSession) IsOwner(_address common.Address) (bool, error) {
	return _DelegateProfile.Contract.IsOwner(&_DelegateProfile.CallOpts, _address)
}

// NumOfFields is a free data retrieval call binding the contract method 0xe6ce112f.
//
// Solidity: function numOfFields() view returns(uint256)
func (_DelegateProfile *DelegateProfileCaller) NumOfFields(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _DelegateProfile.contract.Call(opts, &out, "numOfFields")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// NumOfFields is a free data retrieval call binding the contract method 0xe6ce112f.
//
// Solidity: function numOfFields() view returns(uint256)
func (_DelegateProfile *DelegateProfileSession) NumOfFields() (*big.Int, error) {
	return _DelegateProfile.Contract.NumOfFields(&_DelegateProfile.CallOpts)
}

// NumOfFields is a free data retrieval call binding the contract method 0xe6ce112f.
//
// Solidity: function numOfFields() view returns(uint256)
func (_DelegateProfile *DelegateProfileCallerSession) NumOfFields() (*big.Int, error) {
	return _DelegateProfile.Contract.NumOfFields(&_DelegateProfile.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_DelegateProfile *DelegateProfileCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DelegateProfile.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_DelegateProfile *DelegateProfileSession) Owner() (common.Address, error) {
	return _DelegateProfile.Contract.Owner(&_DelegateProfile.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_DelegateProfile *DelegateProfileCallerSession) Owner() (common.Address, error) {
	return _DelegateProfile.Contract.Owner(&_DelegateProfile.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_DelegateProfile *DelegateProfileCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _DelegateProfile.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_DelegateProfile *DelegateProfileSession) Paused() (bool, error) {
	return _DelegateProfile.Contract.Paused(&_DelegateProfile.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_DelegateProfile *DelegateProfileCallerSession) Paused() (bool, error) {
	return _DelegateProfile.Contract.Paused(&_DelegateProfile.CallOpts)
}

// Register is a free data retrieval call binding the contract method 0x1aa3a008.
//
// Solidity: function register() view returns(address)
func (_DelegateProfile *DelegateProfileCaller) Register(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _DelegateProfile.contract.Call(opts, &out, "register")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Register is a free data retrieval call binding the contract method 0x1aa3a008.
//
// Solidity: function register() view returns(address)
func (_DelegateProfile *DelegateProfileSession) Register() (common.Address, error) {
	return _DelegateProfile.Contract.Register(&_DelegateProfile.CallOpts)
}

// Register is a free data retrieval call binding the contract method 0x1aa3a008.
//
// Solidity: function register() view returns(address)
func (_DelegateProfile *DelegateProfileCallerSession) Register() (common.Address, error) {
	return _DelegateProfile.Contract.Register(&_DelegateProfile.CallOpts)
}

// DeprecateField is a paid mutator transaction binding the contract method 0xe0adf839.
//
// Solidity: function deprecateField(string _name) returns()
func (_DelegateProfile *DelegateProfileTransactor) DeprecateField(opts *bind.TransactOpts, _name string) (*types.Transaction, error) {
	return _DelegateProfile.contract.Transact(opts, "deprecateField", _name)
}

// DeprecateField is a paid mutator transaction binding the contract method 0xe0adf839.
//
// Solidity: function deprecateField(string _name) returns()
func (_DelegateProfile *DelegateProfileSession) DeprecateField(_name string) (*types.Transaction, error) {
	return _DelegateProfile.Contract.DeprecateField(&_DelegateProfile.TransactOpts, _name)
}

// DeprecateField is a paid mutator transaction binding the contract method 0xe0adf839.
//
// Solidity: function deprecateField(string _name) returns()
func (_DelegateProfile *DelegateProfileTransactorSession) DeprecateField(_name string) (*types.Transaction, error) {
	return _DelegateProfile.Contract.DeprecateField(&_DelegateProfile.TransactOpts, _name)
}

// NewField is a paid mutator transaction binding the contract method 0x68beafc8.
//
// Solidity: function newField(string _name, address _verifierAddr) returns()
func (_DelegateProfile *DelegateProfileTransactor) NewField(opts *bind.TransactOpts, _name string, _verifierAddr common.Address) (*types.Transaction, error) {
	return _DelegateProfile.contract.Transact(opts, "newField", _name, _verifierAddr)
}

// NewField is a paid mutator transaction binding the contract method 0x68beafc8.
//
// Solidity: function newField(string _name, address _verifierAddr) returns()
func (_DelegateProfile *DelegateProfileSession) NewField(_name string, _verifierAddr common.Address) (*types.Transaction, error) {
	return _DelegateProfile.Contract.NewField(&_DelegateProfile.TransactOpts, _name, _verifierAddr)
}

// NewField is a paid mutator transaction binding the contract method 0x68beafc8.
//
// Solidity: function newField(string _name, address _verifierAddr) returns()
func (_DelegateProfile *DelegateProfileTransactorSession) NewField(_name string, _verifierAddr common.Address) (*types.Transaction, error) {
	return _DelegateProfile.Contract.NewField(&_DelegateProfile.TransactOpts, _name, _verifierAddr)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_DelegateProfile *DelegateProfileTransactor) Pause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DelegateProfile.contract.Transact(opts, "pause")
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_DelegateProfile *DelegateProfileSession) Pause() (*types.Transaction, error) {
	return _DelegateProfile.Contract.Pause(&_DelegateProfile.TransactOpts)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_DelegateProfile *DelegateProfileTransactorSession) Pause() (*types.Transaction, error) {
	return _DelegateProfile.Contract.Pause(&_DelegateProfile.TransactOpts)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address _newOwner) returns()
func (_DelegateProfile *DelegateProfileTransactor) TransferOwnership(opts *bind.TransactOpts, _newOwner common.Address) (*types.Transaction, error) {
	return _DelegateProfile.contract.Transact(opts, "transferOwnership", _newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address _newOwner) returns()
func (_DelegateProfile *DelegateProfileSession) TransferOwnership(_newOwner common.Address) (*types.Transaction, error) {
	return _DelegateProfile.Contract.TransferOwnership(&_DelegateProfile.TransactOpts, _newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address _newOwner) returns()
func (_DelegateProfile *DelegateProfileTransactorSession) TransferOwnership(_newOwner common.Address) (*types.Transaction, error) {
	return _DelegateProfile.Contract.TransferOwnership(&_DelegateProfile.TransactOpts, _newOwner)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_DelegateProfile *DelegateProfileTransactor) Unpause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DelegateProfile.contract.Transact(opts, "unpause")
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_DelegateProfile *DelegateProfileSession) Unpause() (*types.Transaction, error) {
	return _DelegateProfile.Contract.Unpause(&_DelegateProfile.TransactOpts)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_DelegateProfile *DelegateProfileTransactorSession) Unpause() (*types.Transaction, error) {
	return _DelegateProfile.Contract.Unpause(&_DelegateProfile.TransactOpts)
}

// UpdateProfile is a paid mutator transaction binding the contract method 0x6eeb9b10.
//
// Solidity: function updateProfile(string _name, bytes _value) returns()
func (_DelegateProfile *DelegateProfileTransactor) UpdateProfile(opts *bind.TransactOpts, _name string, _value []byte) (*types.Transaction, error) {
	return _DelegateProfile.contract.Transact(opts, "updateProfile", _name, _value)
}

// UpdateProfile is a paid mutator transaction binding the contract method 0x6eeb9b10.
//
// Solidity: function updateProfile(string _name, bytes _value) returns()
func (_DelegateProfile *DelegateProfileSession) UpdateProfile(_name string, _value []byte) (*types.Transaction, error) {
	return _DelegateProfile.Contract.UpdateProfile(&_DelegateProfile.TransactOpts, _name, _value)
}

// UpdateProfile is a paid mutator transaction binding the contract method 0x6eeb9b10.
//
// Solidity: function updateProfile(string _name, bytes _value) returns()
func (_DelegateProfile *DelegateProfileTransactorSession) UpdateProfile(_name string, _value []byte) (*types.Transaction, error) {
	return _DelegateProfile.Contract.UpdateProfile(&_DelegateProfile.TransactOpts, _name, _value)
}

// UpdateProfileForDelegate is a paid mutator transaction binding the contract method 0x199baa71.
//
// Solidity: function updateProfileForDelegate(address _delegate, string _name, bytes _value) returns()
func (_DelegateProfile *DelegateProfileTransactor) UpdateProfileForDelegate(opts *bind.TransactOpts, _delegate common.Address, _name string, _value []byte) (*types.Transaction, error) {
	return _DelegateProfile.contract.Transact(opts, "updateProfileForDelegate", _delegate, _name, _value)
}

// UpdateProfileForDelegate is a paid mutator transaction binding the contract method 0x199baa71.
//
// Solidity: function updateProfileForDelegate(address _delegate, string _name, bytes _value) returns()
func (_DelegateProfile *DelegateProfileSession) UpdateProfileForDelegate(_delegate common.Address, _name string, _value []byte) (*types.Transaction, error) {
	return _DelegateProfile.Contract.UpdateProfileForDelegate(&_DelegateProfile.TransactOpts, _delegate, _name, _value)
}

// UpdateProfileForDelegate is a paid mutator transaction binding the contract method 0x199baa71.
//
// Solidity: function updateProfileForDelegate(address _delegate, string _name, bytes _value) returns()
func (_DelegateProfile *DelegateProfileTransactorSession) UpdateProfileForDelegate(_delegate common.Address, _name string, _value []byte) (*types.Transaction, error) {
	return _DelegateProfile.Contract.UpdateProfileForDelegate(&_DelegateProfile.TransactOpts, _delegate, _name, _value)
}

// UpdateProfileWithByteCode is a paid mutator transaction binding the contract method 0x37d1f437.
//
// Solidity: function updateProfileWithByteCode(bytes _byteCode) returns()
func (_DelegateProfile *DelegateProfileTransactor) UpdateProfileWithByteCode(opts *bind.TransactOpts, _byteCode []byte) (*types.Transaction, error) {
	return _DelegateProfile.contract.Transact(opts, "updateProfileWithByteCode", _byteCode)
}

// UpdateProfileWithByteCode is a paid mutator transaction binding the contract method 0x37d1f437.
//
// Solidity: function updateProfileWithByteCode(bytes _byteCode) returns()
func (_DelegateProfile *DelegateProfileSession) UpdateProfileWithByteCode(_byteCode []byte) (*types.Transaction, error) {
	return _DelegateProfile.Contract.UpdateProfileWithByteCode(&_DelegateProfile.TransactOpts, _byteCode)
}

// UpdateProfileWithByteCode is a paid mutator transaction binding the contract method 0x37d1f437.
//
// Solidity: function updateProfileWithByteCode(bytes _byteCode) returns()
func (_DelegateProfile *DelegateProfileTransactorSession) UpdateProfileWithByteCode(_byteCode []byte) (*types.Transaction, error) {
	return _DelegateProfile.Contract.UpdateProfileWithByteCode(&_DelegateProfile.TransactOpts, _byteCode)
}

// UpdateProfileWithByteCodeForDelegate is a paid mutator transaction binding the contract method 0xac468ebc.
//
// Solidity: function updateProfileWithByteCodeForDelegate(address _delegate, bytes _byteCode) returns()
func (_DelegateProfile *DelegateProfileTransactor) UpdateProfileWithByteCodeForDelegate(opts *bind.TransactOpts, _delegate common.Address, _byteCode []byte) (*types.Transaction, error) {
	return _DelegateProfile.contract.Transact(opts, "updateProfileWithByteCodeForDelegate", _delegate, _byteCode)
}

// UpdateProfileWithByteCodeForDelegate is a paid mutator transaction binding the contract method 0xac468ebc.
//
// Solidity: function updateProfileWithByteCodeForDelegate(address _delegate, bytes _byteCode) returns()
func (_DelegateProfile *DelegateProfileSession) UpdateProfileWithByteCodeForDelegate(_delegate common.Address, _byteCode []byte) (*types.Transaction, error) {
	return _DelegateProfile.Contract.UpdateProfileWithByteCodeForDelegate(&_DelegateProfile.TransactOpts, _delegate, _byteCode)
}

// UpdateProfileWithByteCodeForDelegate is a paid mutator transaction binding the contract method 0xac468ebc.
//
// Solidity: function updateProfileWithByteCodeForDelegate(address _delegate, bytes _byteCode) returns()
func (_DelegateProfile *DelegateProfileTransactorSession) UpdateProfileWithByteCodeForDelegate(_delegate common.Address, _byteCode []byte) (*types.Transaction, error) {
	return _DelegateProfile.Contract.UpdateProfileWithByteCodeForDelegate(&_DelegateProfile.TransactOpts, _delegate, _byteCode)
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_DelegateProfile *DelegateProfileTransactor) Withdraw(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _DelegateProfile.contract.Transact(opts, "withdraw")
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_DelegateProfile *DelegateProfileSession) Withdraw() (*types.Transaction, error) {
	return _DelegateProfile.Contract.Withdraw(&_DelegateProfile.TransactOpts)
}

// Withdraw is a paid mutator transaction binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() returns()
func (_DelegateProfile *DelegateProfileTransactorSession) Withdraw() (*types.Transaction, error) {
	return _DelegateProfile.Contract.Withdraw(&_DelegateProfile.TransactOpts)
}

// DelegateProfileFeeUpdatedIterator is returned from FilterFeeUpdated and is used to iterate over the raw logs and unpacked data for FeeUpdated events raised by the DelegateProfile contract.
type DelegateProfileFeeUpdatedIterator struct {
	Event *DelegateProfileFeeUpdated // Event containing the contract specifics and raw log

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
func (it *DelegateProfileFeeUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DelegateProfileFeeUpdated)
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
		it.Event = new(DelegateProfileFeeUpdated)
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
func (it *DelegateProfileFeeUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DelegateProfileFeeUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DelegateProfileFeeUpdated represents a FeeUpdated event raised by the DelegateProfile contract.
type DelegateProfileFeeUpdated struct {
	Fee *big.Int
	Raw types.Log // Blockchain specific contextual infos
}

// FilterFeeUpdated is a free log retrieval operation binding the contract event 0x8c4d35e54a3f2ef1134138fd8ea3daee6a3c89e10d2665996babdf70261e2c76.
//
// Solidity: event FeeUpdated(uint256 fee)
func (_DelegateProfile *DelegateProfileFilterer) FilterFeeUpdated(opts *bind.FilterOpts) (*DelegateProfileFeeUpdatedIterator, error) {

	logs, sub, err := _DelegateProfile.contract.FilterLogs(opts, "FeeUpdated")
	if err != nil {
		return nil, err
	}
	return &DelegateProfileFeeUpdatedIterator{contract: _DelegateProfile.contract, event: "FeeUpdated", logs: logs, sub: sub}, nil
}

// WatchFeeUpdated is a free log subscription operation binding the contract event 0x8c4d35e54a3f2ef1134138fd8ea3daee6a3c89e10d2665996babdf70261e2c76.
//
// Solidity: event FeeUpdated(uint256 fee)
func (_DelegateProfile *DelegateProfileFilterer) WatchFeeUpdated(opts *bind.WatchOpts, sink chan<- *DelegateProfileFeeUpdated) (event.Subscription, error) {

	logs, sub, err := _DelegateProfile.contract.WatchLogs(opts, "FeeUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DelegateProfileFeeUpdated)
				if err := _DelegateProfile.contract.UnpackLog(event, "FeeUpdated", log); err != nil {
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

// ParseFeeUpdated is a log parse operation binding the contract event 0x8c4d35e54a3f2ef1134138fd8ea3daee6a3c89e10d2665996babdf70261e2c76.
//
// Solidity: event FeeUpdated(uint256 fee)
func (_DelegateProfile *DelegateProfileFilterer) ParseFeeUpdated(log types.Log) (*DelegateProfileFeeUpdated, error) {
	event := new(DelegateProfileFeeUpdated)
	if err := _DelegateProfile.contract.UnpackLog(event, "FeeUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DelegateProfileFieldDeprecatedIterator is returned from FilterFieldDeprecated and is used to iterate over the raw logs and unpacked data for FieldDeprecated events raised by the DelegateProfile contract.
type DelegateProfileFieldDeprecatedIterator struct {
	Event *DelegateProfileFieldDeprecated // Event containing the contract specifics and raw log

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
func (it *DelegateProfileFieldDeprecatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DelegateProfileFieldDeprecated)
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
		it.Event = new(DelegateProfileFieldDeprecated)
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
func (it *DelegateProfileFieldDeprecatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DelegateProfileFieldDeprecatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DelegateProfileFieldDeprecated represents a FieldDeprecated event raised by the DelegateProfile contract.
type DelegateProfileFieldDeprecated struct {
	Name string
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterFieldDeprecated is a free log retrieval operation binding the contract event 0xf47b35d35c737e18368ebfa5496bc97dabcea3e7b0075269da84fc32d0f201b8.
//
// Solidity: event FieldDeprecated(string name)
func (_DelegateProfile *DelegateProfileFilterer) FilterFieldDeprecated(opts *bind.FilterOpts) (*DelegateProfileFieldDeprecatedIterator, error) {

	logs, sub, err := _DelegateProfile.contract.FilterLogs(opts, "FieldDeprecated")
	if err != nil {
		return nil, err
	}
	return &DelegateProfileFieldDeprecatedIterator{contract: _DelegateProfile.contract, event: "FieldDeprecated", logs: logs, sub: sub}, nil
}

// WatchFieldDeprecated is a free log subscription operation binding the contract event 0xf47b35d35c737e18368ebfa5496bc97dabcea3e7b0075269da84fc32d0f201b8.
//
// Solidity: event FieldDeprecated(string name)
func (_DelegateProfile *DelegateProfileFilterer) WatchFieldDeprecated(opts *bind.WatchOpts, sink chan<- *DelegateProfileFieldDeprecated) (event.Subscription, error) {

	logs, sub, err := _DelegateProfile.contract.WatchLogs(opts, "FieldDeprecated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DelegateProfileFieldDeprecated)
				if err := _DelegateProfile.contract.UnpackLog(event, "FieldDeprecated", log); err != nil {
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

// ParseFieldDeprecated is a log parse operation binding the contract event 0xf47b35d35c737e18368ebfa5496bc97dabcea3e7b0075269da84fc32d0f201b8.
//
// Solidity: event FieldDeprecated(string name)
func (_DelegateProfile *DelegateProfileFilterer) ParseFieldDeprecated(log types.Log) (*DelegateProfileFieldDeprecated, error) {
	event := new(DelegateProfileFieldDeprecated)
	if err := _DelegateProfile.contract.UnpackLog(event, "FieldDeprecated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DelegateProfileNewFieldIterator is returned from FilterNewField and is used to iterate over the raw logs and unpacked data for NewField events raised by the DelegateProfile contract.
type DelegateProfileNewFieldIterator struct {
	Event *DelegateProfileNewField // Event containing the contract specifics and raw log

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
func (it *DelegateProfileNewFieldIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DelegateProfileNewField)
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
		it.Event = new(DelegateProfileNewField)
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
func (it *DelegateProfileNewFieldIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DelegateProfileNewFieldIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DelegateProfileNewField represents a NewField event raised by the DelegateProfile contract.
type DelegateProfileNewField struct {
	Name string
	Raw  types.Log // Blockchain specific contextual infos
}

// FilterNewField is a free log retrieval operation binding the contract event 0x53096991d49a1876b3be4d7f3d107f7f92043e0fceec1e81b5ba38841d78123b.
//
// Solidity: event NewField(string name)
func (_DelegateProfile *DelegateProfileFilterer) FilterNewField(opts *bind.FilterOpts) (*DelegateProfileNewFieldIterator, error) {

	logs, sub, err := _DelegateProfile.contract.FilterLogs(opts, "NewField")
	if err != nil {
		return nil, err
	}
	return &DelegateProfileNewFieldIterator{contract: _DelegateProfile.contract, event: "NewField", logs: logs, sub: sub}, nil
}

// WatchNewField is a free log subscription operation binding the contract event 0x53096991d49a1876b3be4d7f3d107f7f92043e0fceec1e81b5ba38841d78123b.
//
// Solidity: event NewField(string name)
func (_DelegateProfile *DelegateProfileFilterer) WatchNewField(opts *bind.WatchOpts, sink chan<- *DelegateProfileNewField) (event.Subscription, error) {

	logs, sub, err := _DelegateProfile.contract.WatchLogs(opts, "NewField")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DelegateProfileNewField)
				if err := _DelegateProfile.contract.UnpackLog(event, "NewField", log); err != nil {
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

// ParseNewField is a log parse operation binding the contract event 0x53096991d49a1876b3be4d7f3d107f7f92043e0fceec1e81b5ba38841d78123b.
//
// Solidity: event NewField(string name)
func (_DelegateProfile *DelegateProfileFilterer) ParseNewField(log types.Log) (*DelegateProfileNewField, error) {
	event := new(DelegateProfileNewField)
	if err := _DelegateProfile.contract.UnpackLog(event, "NewField", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DelegateProfilePauseIterator is returned from FilterPause and is used to iterate over the raw logs and unpacked data for Pause events raised by the DelegateProfile contract.
type DelegateProfilePauseIterator struct {
	Event *DelegateProfilePause // Event containing the contract specifics and raw log

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
func (it *DelegateProfilePauseIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DelegateProfilePause)
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
		it.Event = new(DelegateProfilePause)
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
func (it *DelegateProfilePauseIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DelegateProfilePauseIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DelegateProfilePause represents a Pause event raised by the DelegateProfile contract.
type DelegateProfilePause struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterPause is a free log retrieval operation binding the contract event 0x6985a02210a168e66602d3235cb6db0e70f92b3ba4d376a33c0f3d9434bff625.
//
// Solidity: event Pause()
func (_DelegateProfile *DelegateProfileFilterer) FilterPause(opts *bind.FilterOpts) (*DelegateProfilePauseIterator, error) {

	logs, sub, err := _DelegateProfile.contract.FilterLogs(opts, "Pause")
	if err != nil {
		return nil, err
	}
	return &DelegateProfilePauseIterator{contract: _DelegateProfile.contract, event: "Pause", logs: logs, sub: sub}, nil
}

// WatchPause is a free log subscription operation binding the contract event 0x6985a02210a168e66602d3235cb6db0e70f92b3ba4d376a33c0f3d9434bff625.
//
// Solidity: event Pause()
func (_DelegateProfile *DelegateProfileFilterer) WatchPause(opts *bind.WatchOpts, sink chan<- *DelegateProfilePause) (event.Subscription, error) {

	logs, sub, err := _DelegateProfile.contract.WatchLogs(opts, "Pause")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DelegateProfilePause)
				if err := _DelegateProfile.contract.UnpackLog(event, "Pause", log); err != nil {
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

// ParsePause is a log parse operation binding the contract event 0x6985a02210a168e66602d3235cb6db0e70f92b3ba4d376a33c0f3d9434bff625.
//
// Solidity: event Pause()
func (_DelegateProfile *DelegateProfileFilterer) ParsePause(log types.Log) (*DelegateProfilePause, error) {
	event := new(DelegateProfilePause)
	if err := _DelegateProfile.contract.UnpackLog(event, "Pause", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DelegateProfileProfileUpdatedIterator is returned from FilterProfileUpdated and is used to iterate over the raw logs and unpacked data for ProfileUpdated events raised by the DelegateProfile contract.
type DelegateProfileProfileUpdatedIterator struct {
	Event *DelegateProfileProfileUpdated // Event containing the contract specifics and raw log

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
func (it *DelegateProfileProfileUpdatedIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DelegateProfileProfileUpdated)
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
		it.Event = new(DelegateProfileProfileUpdated)
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
func (it *DelegateProfileProfileUpdatedIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DelegateProfileProfileUpdatedIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DelegateProfileProfileUpdated represents a ProfileUpdated event raised by the DelegateProfile contract.
type DelegateProfileProfileUpdated struct {
	Delegate common.Address
	Name     string
	Value    []byte
	Raw      types.Log // Blockchain specific contextual infos
}

// FilterProfileUpdated is a free log retrieval operation binding the contract event 0x217aa5ef0b78f028d51fd573433bdbe2daf6f8505e6a71f3af1393c8440b341b.
//
// Solidity: event ProfileUpdated(address delegate, string name, bytes value)
func (_DelegateProfile *DelegateProfileFilterer) FilterProfileUpdated(opts *bind.FilterOpts) (*DelegateProfileProfileUpdatedIterator, error) {

	logs, sub, err := _DelegateProfile.contract.FilterLogs(opts, "ProfileUpdated")
	if err != nil {
		return nil, err
	}
	return &DelegateProfileProfileUpdatedIterator{contract: _DelegateProfile.contract, event: "ProfileUpdated", logs: logs, sub: sub}, nil
}

// WatchProfileUpdated is a free log subscription operation binding the contract event 0x217aa5ef0b78f028d51fd573433bdbe2daf6f8505e6a71f3af1393c8440b341b.
//
// Solidity: event ProfileUpdated(address delegate, string name, bytes value)
func (_DelegateProfile *DelegateProfileFilterer) WatchProfileUpdated(opts *bind.WatchOpts, sink chan<- *DelegateProfileProfileUpdated) (event.Subscription, error) {

	logs, sub, err := _DelegateProfile.contract.WatchLogs(opts, "ProfileUpdated")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DelegateProfileProfileUpdated)
				if err := _DelegateProfile.contract.UnpackLog(event, "ProfileUpdated", log); err != nil {
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

// ParseProfileUpdated is a log parse operation binding the contract event 0x217aa5ef0b78f028d51fd573433bdbe2daf6f8505e6a71f3af1393c8440b341b.
//
// Solidity: event ProfileUpdated(address delegate, string name, bytes value)
func (_DelegateProfile *DelegateProfileFilterer) ParseProfileUpdated(log types.Log) (*DelegateProfileProfileUpdated, error) {
	event := new(DelegateProfileProfileUpdated)
	if err := _DelegateProfile.contract.UnpackLog(event, "ProfileUpdated", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// DelegateProfileUnpauseIterator is returned from FilterUnpause and is used to iterate over the raw logs and unpacked data for Unpause events raised by the DelegateProfile contract.
type DelegateProfileUnpauseIterator struct {
	Event *DelegateProfileUnpause // Event containing the contract specifics and raw log

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
func (it *DelegateProfileUnpauseIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(DelegateProfileUnpause)
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
		it.Event = new(DelegateProfileUnpause)
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
func (it *DelegateProfileUnpauseIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *DelegateProfileUnpauseIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// DelegateProfileUnpause represents a Unpause event raised by the DelegateProfile contract.
type DelegateProfileUnpause struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterUnpause is a free log retrieval operation binding the contract event 0x7805862f689e2f13df9f062ff482ad3ad112aca9e0847911ed832e158c525b33.
//
// Solidity: event Unpause()
func (_DelegateProfile *DelegateProfileFilterer) FilterUnpause(opts *bind.FilterOpts) (*DelegateProfileUnpauseIterator, error) {

	logs, sub, err := _DelegateProfile.contract.FilterLogs(opts, "Unpause")
	if err != nil {
		return nil, err
	}
	return &DelegateProfileUnpauseIterator{contract: _DelegateProfile.contract, event: "Unpause", logs: logs, sub: sub}, nil
}

// WatchUnpause is a free log subscription operation binding the contract event 0x7805862f689e2f13df9f062ff482ad3ad112aca9e0847911ed832e158c525b33.
//
// Solidity: event Unpause()
func (_DelegateProfile *DelegateProfileFilterer) WatchUnpause(opts *bind.WatchOpts, sink chan<- *DelegateProfileUnpause) (event.Subscription, error) {

	logs, sub, err := _DelegateProfile.contract.WatchLogs(opts, "Unpause")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(DelegateProfileUnpause)
				if err := _DelegateProfile.contract.UnpackLog(event, "Unpause", log); err != nil {
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

// ParseUnpause is a log parse operation binding the contract event 0x7805862f689e2f13df9f062ff482ad3ad112aca9e0847911ed832e158c525b33.
//
// Solidity: event Unpause()
func (_DelegateProfile *DelegateProfileFilterer) ParseUnpause(log types.Log) (*DelegateProfileUnpause, error) {
	event := new(DelegateProfileUnpause)
	if err := _DelegateProfile.contract.UnpackLog(event, "Unpause", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// OwnableMetaData contains all meta data concerning the Ownable contract.
var OwnableMetaData = &bind.MetaData{
	ABI: "[{\"constant\":true,\"inputs\":[{\"name\":\"_address\",\"type\":\"address\"}],\"name\":\"isOwner\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"}]",
	Sigs: map[string]string{
		"2f54bf6e": "isOwner(address)",
		"8da5cb5b": "owner()",
		"f2fde38b": "transferOwnership(address)",
	},
	Bin: "0x608060405234801561001057600080fd5b5060008054600160a060020a031916331790556101c4806100326000396000f3006080604052600436106100565763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416632f54bf6e811461005b5780638da5cb5b1461009d578063f2fde38b146100db575b600080fd5b34801561006757600080fd5b5061008973ffffffffffffffffffffffffffffffffffffffff6004351661010b565b604080519115158252519081900360200190f35b3480156100a957600080fd5b506100b261012c565b6040805173ffffffffffffffffffffffffffffffffffffffff9092168252519081900360200190f35b3480156100e757600080fd5b5061010973ffffffffffffffffffffffffffffffffffffffff60043516610148565b005b60005473ffffffffffffffffffffffffffffffffffffffff91821691161490565b60005473ffffffffffffffffffffffffffffffffffffffff1681565b6101513361010b565b151561015c57600080fd5b6000805473ffffffffffffffffffffffffffffffffffffffff191673ffffffffffffffffffffffffffffffffffffffff929092169190911790555600a165627a7a72305820bc80d1e4d9cdf119a875cba83d74f915a782daaa88f8e73adf3d68d787002ac10029",
}

// OwnableABI is the input ABI used to generate the binding from.
// Deprecated: Use OwnableMetaData.ABI instead.
var OwnableABI = OwnableMetaData.ABI

// Deprecated: Use OwnableMetaData.Sigs instead.
// OwnableFuncSigs maps the 4-byte function signature to its string representation.
var OwnableFuncSigs = OwnableMetaData.Sigs

// OwnableBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use OwnableMetaData.Bin instead.
var OwnableBin = OwnableMetaData.Bin

// DeployOwnable deploys a new Ethereum contract, binding an instance of Ownable to it.
func DeployOwnable(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Ownable, error) {
	parsed, err := OwnableMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(OwnableBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Ownable{OwnableCaller: OwnableCaller{contract: contract}, OwnableTransactor: OwnableTransactor{contract: contract}, OwnableFilterer: OwnableFilterer{contract: contract}}, nil
}

// Ownable is an auto generated Go binding around an Ethereum contract.
type Ownable struct {
	OwnableCaller     // Read-only binding to the contract
	OwnableTransactor // Write-only binding to the contract
	OwnableFilterer   // Log filterer for contract events
}

// OwnableCaller is an auto generated read-only Go binding around an Ethereum contract.
type OwnableCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OwnableTransactor is an auto generated write-only Go binding around an Ethereum contract.
type OwnableTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OwnableFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type OwnableFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// OwnableSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type OwnableSession struct {
	Contract     *Ownable          // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// OwnableCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type OwnableCallerSession struct {
	Contract *OwnableCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts  // Call options to use throughout this session
}

// OwnableTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type OwnableTransactorSession struct {
	Contract     *OwnableTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts  // Transaction auth options to use throughout this session
}

// OwnableRaw is an auto generated low-level Go binding around an Ethereum contract.
type OwnableRaw struct {
	Contract *Ownable // Generic contract binding to access the raw methods on
}

// OwnableCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type OwnableCallerRaw struct {
	Contract *OwnableCaller // Generic read-only contract binding to access the raw methods on
}

// OwnableTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type OwnableTransactorRaw struct {
	Contract *OwnableTransactor // Generic write-only contract binding to access the raw methods on
}

// NewOwnable creates a new instance of Ownable, bound to a specific deployed contract.
func NewOwnable(address common.Address, backend bind.ContractBackend) (*Ownable, error) {
	contract, err := bindOwnable(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Ownable{OwnableCaller: OwnableCaller{contract: contract}, OwnableTransactor: OwnableTransactor{contract: contract}, OwnableFilterer: OwnableFilterer{contract: contract}}, nil
}

// NewOwnableCaller creates a new read-only instance of Ownable, bound to a specific deployed contract.
func NewOwnableCaller(address common.Address, caller bind.ContractCaller) (*OwnableCaller, error) {
	contract, err := bindOwnable(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &OwnableCaller{contract: contract}, nil
}

// NewOwnableTransactor creates a new write-only instance of Ownable, bound to a specific deployed contract.
func NewOwnableTransactor(address common.Address, transactor bind.ContractTransactor) (*OwnableTransactor, error) {
	contract, err := bindOwnable(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &OwnableTransactor{contract: contract}, nil
}

// NewOwnableFilterer creates a new log filterer instance of Ownable, bound to a specific deployed contract.
func NewOwnableFilterer(address common.Address, filterer bind.ContractFilterer) (*OwnableFilterer, error) {
	contract, err := bindOwnable(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &OwnableFilterer{contract: contract}, nil
}

// bindOwnable binds a generic wrapper to an already deployed contract.
func bindOwnable(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(OwnableABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Ownable *OwnableRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Ownable.Contract.OwnableCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Ownable *OwnableRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Ownable.Contract.OwnableTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Ownable *OwnableRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Ownable.Contract.OwnableTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Ownable *OwnableCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Ownable.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Ownable *OwnableTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Ownable.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Ownable *OwnableTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Ownable.Contract.contract.Transact(opts, method, params...)
}

// IsOwner is a free data retrieval call binding the contract method 0x2f54bf6e.
//
// Solidity: function isOwner(address _address) view returns(bool)
func (_Ownable *OwnableCaller) IsOwner(opts *bind.CallOpts, _address common.Address) (bool, error) {
	var out []interface{}
	err := _Ownable.contract.Call(opts, &out, "isOwner", _address)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsOwner is a free data retrieval call binding the contract method 0x2f54bf6e.
//
// Solidity: function isOwner(address _address) view returns(bool)
func (_Ownable *OwnableSession) IsOwner(_address common.Address) (bool, error) {
	return _Ownable.Contract.IsOwner(&_Ownable.CallOpts, _address)
}

// IsOwner is a free data retrieval call binding the contract method 0x2f54bf6e.
//
// Solidity: function isOwner(address _address) view returns(bool)
func (_Ownable *OwnableCallerSession) IsOwner(_address common.Address) (bool, error) {
	return _Ownable.Contract.IsOwner(&_Ownable.CallOpts, _address)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Ownable *OwnableCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Ownable.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Ownable *OwnableSession) Owner() (common.Address, error) {
	return _Ownable.Contract.Owner(&_Ownable.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Ownable *OwnableCallerSession) Owner() (common.Address, error) {
	return _Ownable.Contract.Owner(&_Ownable.CallOpts)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address _newOwner) returns()
func (_Ownable *OwnableTransactor) TransferOwnership(opts *bind.TransactOpts, _newOwner common.Address) (*types.Transaction, error) {
	return _Ownable.contract.Transact(opts, "transferOwnership", _newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address _newOwner) returns()
func (_Ownable *OwnableSession) TransferOwnership(_newOwner common.Address) (*types.Transaction, error) {
	return _Ownable.Contract.TransferOwnership(&_Ownable.TransactOpts, _newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address _newOwner) returns()
func (_Ownable *OwnableTransactorSession) TransferOwnership(_newOwner common.Address) (*types.Transaction, error) {
	return _Ownable.Contract.TransferOwnership(&_Ownable.TransactOpts, _newOwner)
}

// PausableMetaData contains all meta data concerning the Pausable contract.
var PausableMetaData = &bind.MetaData{
	ABI: "[{\"constant\":true,\"inputs\":[{\"name\":\"_address\",\"type\":\"address\"}],\"name\":\"isOwner\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"unpause\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"paused\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[],\"name\":\"pause\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"name\":\"\",\"type\":\"address\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"_newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"anonymous\":false,\"inputs\":[],\"name\":\"Pause\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[],\"name\":\"Unpause\",\"type\":\"event\"}]",
	Sigs: map[string]string{
		"2f54bf6e": "isOwner(address)",
		"8da5cb5b": "owner()",
		"8456cb59": "pause()",
		"5c975abb": "paused()",
		"f2fde38b": "transferOwnership(address)",
		"3f4ba83a": "unpause()",
	},
	Bin: "0x608060405260008054600160a860020a03191633179055610363806100256000396000f3006080604052600436106100775763ffffffff7c01000000000000000000000000000000000000000000000000000000006000350416632f54bf6e811461007c5780633f4ba83a146100be5780635c975abb146100d55780638456cb59146100ea5780638da5cb5b146100ff578063f2fde38b1461013d575b600080fd5b34801561008857600080fd5b506100aa73ffffffffffffffffffffffffffffffffffffffff6004351661016b565b604080519115158252519081900360200190f35b3480156100ca57600080fd5b506100d361018c565b005b3480156100e157600080fd5b506100aa610210565b3480156100f657600080fd5b506100d3610231565b34801561010b57600080fd5b506101146102cb565b6040805173ffffffffffffffffffffffffffffffffffffffff9092168252519081900360200190f35b34801561014957600080fd5b506100d373ffffffffffffffffffffffffffffffffffffffff600435166102e7565b60005473ffffffffffffffffffffffffffffffffffffffff91821691161490565b6101953361016b565b15156101a057600080fd5b60005474010000000000000000000000000000000000000000900460ff1615156101c957600080fd5b6000805474ff0000000000000000000000000000000000000000191681556040517f7805862f689e2f13df9f062ff482ad3ad112aca9e0847911ed832e158c525b339190a1565b60005474010000000000000000000000000000000000000000900460ff1681565b61023a3361016b565b151561024557600080fd5b60005474010000000000000000000000000000000000000000900460ff161561026d57600080fd5b6000805474ff00000000000000000000000000000000000000001916740100000000000000000000000000000000000000001781556040517f6985a02210a168e66602d3235cb6db0e70f92b3ba4d376a33c0f3d9434bff6259190a1565b60005473ffffffffffffffffffffffffffffffffffffffff1681565b6102f03361016b565b15156102fb57600080fd5b6000805473ffffffffffffffffffffffffffffffffffffffff191673ffffffffffffffffffffffffffffffffffffffff929092169190911790555600a165627a7a72305820fee0b0c512d6d4bb7d1fd0b05a522cdf0d9ea479d7101b4d7eb4f888391a6cf70029",
}

// PausableABI is the input ABI used to generate the binding from.
// Deprecated: Use PausableMetaData.ABI instead.
var PausableABI = PausableMetaData.ABI

// Deprecated: Use PausableMetaData.Sigs instead.
// PausableFuncSigs maps the 4-byte function signature to its string representation.
var PausableFuncSigs = PausableMetaData.Sigs

// PausableBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use PausableMetaData.Bin instead.
var PausableBin = PausableMetaData.Bin

// DeployPausable deploys a new Ethereum contract, binding an instance of Pausable to it.
func DeployPausable(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Pausable, error) {
	parsed, err := PausableMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(PausableBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Pausable{PausableCaller: PausableCaller{contract: contract}, PausableTransactor: PausableTransactor{contract: contract}, PausableFilterer: PausableFilterer{contract: contract}}, nil
}

// Pausable is an auto generated Go binding around an Ethereum contract.
type Pausable struct {
	PausableCaller     // Read-only binding to the contract
	PausableTransactor // Write-only binding to the contract
	PausableFilterer   // Log filterer for contract events
}

// PausableCaller is an auto generated read-only Go binding around an Ethereum contract.
type PausableCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PausableTransactor is an auto generated write-only Go binding around an Ethereum contract.
type PausableTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PausableFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type PausableFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// PausableSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type PausableSession struct {
	Contract     *Pausable         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// PausableCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type PausableCallerSession struct {
	Contract *PausableCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// PausableTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type PausableTransactorSession struct {
	Contract     *PausableTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// PausableRaw is an auto generated low-level Go binding around an Ethereum contract.
type PausableRaw struct {
	Contract *Pausable // Generic contract binding to access the raw methods on
}

// PausableCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type PausableCallerRaw struct {
	Contract *PausableCaller // Generic read-only contract binding to access the raw methods on
}

// PausableTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type PausableTransactorRaw struct {
	Contract *PausableTransactor // Generic write-only contract binding to access the raw methods on
}

// NewPausable creates a new instance of Pausable, bound to a specific deployed contract.
func NewPausable(address common.Address, backend bind.ContractBackend) (*Pausable, error) {
	contract, err := bindPausable(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Pausable{PausableCaller: PausableCaller{contract: contract}, PausableTransactor: PausableTransactor{contract: contract}, PausableFilterer: PausableFilterer{contract: contract}}, nil
}

// NewPausableCaller creates a new read-only instance of Pausable, bound to a specific deployed contract.
func NewPausableCaller(address common.Address, caller bind.ContractCaller) (*PausableCaller, error) {
	contract, err := bindPausable(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &PausableCaller{contract: contract}, nil
}

// NewPausableTransactor creates a new write-only instance of Pausable, bound to a specific deployed contract.
func NewPausableTransactor(address common.Address, transactor bind.ContractTransactor) (*PausableTransactor, error) {
	contract, err := bindPausable(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &PausableTransactor{contract: contract}, nil
}

// NewPausableFilterer creates a new log filterer instance of Pausable, bound to a specific deployed contract.
func NewPausableFilterer(address common.Address, filterer bind.ContractFilterer) (*PausableFilterer, error) {
	contract, err := bindPausable(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &PausableFilterer{contract: contract}, nil
}

// bindPausable binds a generic wrapper to an already deployed contract.
func bindPausable(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(PausableABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Pausable *PausableRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Pausable.Contract.PausableCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Pausable *PausableRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Pausable.Contract.PausableTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Pausable *PausableRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Pausable.Contract.PausableTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Pausable *PausableCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Pausable.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Pausable *PausableTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Pausable.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Pausable *PausableTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Pausable.Contract.contract.Transact(opts, method, params...)
}

// IsOwner is a free data retrieval call binding the contract method 0x2f54bf6e.
//
// Solidity: function isOwner(address _address) view returns(bool)
func (_Pausable *PausableCaller) IsOwner(opts *bind.CallOpts, _address common.Address) (bool, error) {
	var out []interface{}
	err := _Pausable.contract.Call(opts, &out, "isOwner", _address)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsOwner is a free data retrieval call binding the contract method 0x2f54bf6e.
//
// Solidity: function isOwner(address _address) view returns(bool)
func (_Pausable *PausableSession) IsOwner(_address common.Address) (bool, error) {
	return _Pausable.Contract.IsOwner(&_Pausable.CallOpts, _address)
}

// IsOwner is a free data retrieval call binding the contract method 0x2f54bf6e.
//
// Solidity: function isOwner(address _address) view returns(bool)
func (_Pausable *PausableCallerSession) IsOwner(_address common.Address) (bool, error) {
	return _Pausable.Contract.IsOwner(&_Pausable.CallOpts, _address)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Pausable *PausableCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Pausable.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Pausable *PausableSession) Owner() (common.Address, error) {
	return _Pausable.Contract.Owner(&_Pausable.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_Pausable *PausableCallerSession) Owner() (common.Address, error) {
	return _Pausable.Contract.Owner(&_Pausable.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_Pausable *PausableCaller) Paused(opts *bind.CallOpts) (bool, error) {
	var out []interface{}
	err := _Pausable.contract.Call(opts, &out, "paused")

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_Pausable *PausableSession) Paused() (bool, error) {
	return _Pausable.Contract.Paused(&_Pausable.CallOpts)
}

// Paused is a free data retrieval call binding the contract method 0x5c975abb.
//
// Solidity: function paused() view returns(bool)
func (_Pausable *PausableCallerSession) Paused() (bool, error) {
	return _Pausable.Contract.Paused(&_Pausable.CallOpts)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_Pausable *PausableTransactor) Pause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Pausable.contract.Transact(opts, "pause")
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_Pausable *PausableSession) Pause() (*types.Transaction, error) {
	return _Pausable.Contract.Pause(&_Pausable.TransactOpts)
}

// Pause is a paid mutator transaction binding the contract method 0x8456cb59.
//
// Solidity: function pause() returns()
func (_Pausable *PausableTransactorSession) Pause() (*types.Transaction, error) {
	return _Pausable.Contract.Pause(&_Pausable.TransactOpts)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address _newOwner) returns()
func (_Pausable *PausableTransactor) TransferOwnership(opts *bind.TransactOpts, _newOwner common.Address) (*types.Transaction, error) {
	return _Pausable.contract.Transact(opts, "transferOwnership", _newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address _newOwner) returns()
func (_Pausable *PausableSession) TransferOwnership(_newOwner common.Address) (*types.Transaction, error) {
	return _Pausable.Contract.TransferOwnership(&_Pausable.TransactOpts, _newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address _newOwner) returns()
func (_Pausable *PausableTransactorSession) TransferOwnership(_newOwner common.Address) (*types.Transaction, error) {
	return _Pausable.Contract.TransferOwnership(&_Pausable.TransactOpts, _newOwner)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_Pausable *PausableTransactor) Unpause(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Pausable.contract.Transact(opts, "unpause")
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_Pausable *PausableSession) Unpause() (*types.Transaction, error) {
	return _Pausable.Contract.Unpause(&_Pausable.TransactOpts)
}

// Unpause is a paid mutator transaction binding the contract method 0x3f4ba83a.
//
// Solidity: function unpause() returns()
func (_Pausable *PausableTransactorSession) Unpause() (*types.Transaction, error) {
	return _Pausable.Contract.Unpause(&_Pausable.TransactOpts)
}

// PausablePauseIterator is returned from FilterPause and is used to iterate over the raw logs and unpacked data for Pause events raised by the Pausable contract.
type PausablePauseIterator struct {
	Event *PausablePause // Event containing the contract specifics and raw log

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
func (it *PausablePauseIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PausablePause)
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
		it.Event = new(PausablePause)
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
func (it *PausablePauseIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PausablePauseIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PausablePause represents a Pause event raised by the Pausable contract.
type PausablePause struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterPause is a free log retrieval operation binding the contract event 0x6985a02210a168e66602d3235cb6db0e70f92b3ba4d376a33c0f3d9434bff625.
//
// Solidity: event Pause()
func (_Pausable *PausableFilterer) FilterPause(opts *bind.FilterOpts) (*PausablePauseIterator, error) {

	logs, sub, err := _Pausable.contract.FilterLogs(opts, "Pause")
	if err != nil {
		return nil, err
	}
	return &PausablePauseIterator{contract: _Pausable.contract, event: "Pause", logs: logs, sub: sub}, nil
}

// WatchPause is a free log subscription operation binding the contract event 0x6985a02210a168e66602d3235cb6db0e70f92b3ba4d376a33c0f3d9434bff625.
//
// Solidity: event Pause()
func (_Pausable *PausableFilterer) WatchPause(opts *bind.WatchOpts, sink chan<- *PausablePause) (event.Subscription, error) {

	logs, sub, err := _Pausable.contract.WatchLogs(opts, "Pause")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PausablePause)
				if err := _Pausable.contract.UnpackLog(event, "Pause", log); err != nil {
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

// ParsePause is a log parse operation binding the contract event 0x6985a02210a168e66602d3235cb6db0e70f92b3ba4d376a33c0f3d9434bff625.
//
// Solidity: event Pause()
func (_Pausable *PausableFilterer) ParsePause(log types.Log) (*PausablePause, error) {
	event := new(PausablePause)
	if err := _Pausable.contract.UnpackLog(event, "Pause", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// PausableUnpauseIterator is returned from FilterUnpause and is used to iterate over the raw logs and unpacked data for Unpause events raised by the Pausable contract.
type PausableUnpauseIterator struct {
	Event *PausableUnpause // Event containing the contract specifics and raw log

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
func (it *PausableUnpauseIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(PausableUnpause)
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
		it.Event = new(PausableUnpause)
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
func (it *PausableUnpauseIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *PausableUnpauseIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// PausableUnpause represents a Unpause event raised by the Pausable contract.
type PausableUnpause struct {
	Raw types.Log // Blockchain specific contextual infos
}

// FilterUnpause is a free log retrieval operation binding the contract event 0x7805862f689e2f13df9f062ff482ad3ad112aca9e0847911ed832e158c525b33.
//
// Solidity: event Unpause()
func (_Pausable *PausableFilterer) FilterUnpause(opts *bind.FilterOpts) (*PausableUnpauseIterator, error) {

	logs, sub, err := _Pausable.contract.FilterLogs(opts, "Unpause")
	if err != nil {
		return nil, err
	}
	return &PausableUnpauseIterator{contract: _Pausable.contract, event: "Unpause", logs: logs, sub: sub}, nil
}

// WatchUnpause is a free log subscription operation binding the contract event 0x7805862f689e2f13df9f062ff482ad3ad112aca9e0847911ed832e158c525b33.
//
// Solidity: event Unpause()
func (_Pausable *PausableFilterer) WatchUnpause(opts *bind.WatchOpts, sink chan<- *PausableUnpause) (event.Subscription, error) {

	logs, sub, err := _Pausable.contract.WatchLogs(opts, "Unpause")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(PausableUnpause)
				if err := _Pausable.contract.UnpackLog(event, "Unpause", log); err != nil {
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

// ParseUnpause is a log parse operation binding the contract event 0x7805862f689e2f13df9f062ff482ad3ad112aca9e0847911ed832e158c525b33.
//
// Solidity: event Unpause()
func (_Pausable *PausableFilterer) ParseUnpause(log types.Log) (*PausableUnpause, error) {
	event := new(PausableUnpause)
	if err := _Pausable.contract.UnpackLog(event, "Unpause", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RegisterMetaData contains all meta data concerning the Register contract.
var RegisterMetaData = &bind.MetaData{
	ABI: "[{\"constant\":true,\"inputs\":[{\"name\":\"\",\"type\":\"address\"}],\"name\":\"addrToIdx\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Sigs: map[string]string{
		"5f609695": "addrToIdx(address)",
	},
}

// RegisterABI is the input ABI used to generate the binding from.
// Deprecated: Use RegisterMetaData.ABI instead.
var RegisterABI = RegisterMetaData.ABI

// Deprecated: Use RegisterMetaData.Sigs instead.
// RegisterFuncSigs maps the 4-byte function signature to its string representation.
var RegisterFuncSigs = RegisterMetaData.Sigs

// Register is an auto generated Go binding around an Ethereum contract.
type Register struct {
	RegisterCaller     // Read-only binding to the contract
	RegisterTransactor // Write-only binding to the contract
	RegisterFilterer   // Log filterer for contract events
}

// RegisterCaller is an auto generated read-only Go binding around an Ethereum contract.
type RegisterCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RegisterTransactor is an auto generated write-only Go binding around an Ethereum contract.
type RegisterTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RegisterFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type RegisterFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RegisterSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type RegisterSession struct {
	Contract     *Register         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// RegisterCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type RegisterCallerSession struct {
	Contract *RegisterCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// RegisterTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type RegisterTransactorSession struct {
	Contract     *RegisterTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// RegisterRaw is an auto generated low-level Go binding around an Ethereum contract.
type RegisterRaw struct {
	Contract *Register // Generic contract binding to access the raw methods on
}

// RegisterCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type RegisterCallerRaw struct {
	Contract *RegisterCaller // Generic read-only contract binding to access the raw methods on
}

// RegisterTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type RegisterTransactorRaw struct {
	Contract *RegisterTransactor // Generic write-only contract binding to access the raw methods on
}

// NewRegister creates a new instance of Register, bound to a specific deployed contract.
func NewRegister(address common.Address, backend bind.ContractBackend) (*Register, error) {
	contract, err := bindRegister(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Register{RegisterCaller: RegisterCaller{contract: contract}, RegisterTransactor: RegisterTransactor{contract: contract}, RegisterFilterer: RegisterFilterer{contract: contract}}, nil
}

// NewRegisterCaller creates a new read-only instance of Register, bound to a specific deployed contract.
func NewRegisterCaller(address common.Address, caller bind.ContractCaller) (*RegisterCaller, error) {
	contract, err := bindRegister(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &RegisterCaller{contract: contract}, nil
}

// NewRegisterTransactor creates a new write-only instance of Register, bound to a specific deployed contract.
func NewRegisterTransactor(address common.Address, transactor bind.ContractTransactor) (*RegisterTransactor, error) {
	contract, err := bindRegister(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &RegisterTransactor{contract: contract}, nil
}

// NewRegisterFilterer creates a new log filterer instance of Register, bound to a specific deployed contract.
func NewRegisterFilterer(address common.Address, filterer bind.ContractFilterer) (*RegisterFilterer, error) {
	contract, err := bindRegister(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &RegisterFilterer{contract: contract}, nil
}

// bindRegister binds a generic wrapper to an already deployed contract.
func bindRegister(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(RegisterABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Register *RegisterRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Register.Contract.RegisterCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Register *RegisterRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Register.Contract.RegisterTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Register *RegisterRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Register.Contract.RegisterTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Register *RegisterCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Register.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Register *RegisterTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Register.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Register *RegisterTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Register.Contract.contract.Transact(opts, method, params...)
}

// AddrToIdx is a free data retrieval call binding the contract method 0x5f609695.
//
// Solidity: function addrToIdx(address ) view returns(uint256)
func (_Register *RegisterCaller) AddrToIdx(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Register.contract.Call(opts, &out, "addrToIdx", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// AddrToIdx is a free data retrieval call binding the contract method 0x5f609695.
//
// Solidity: function addrToIdx(address ) view returns(uint256)
func (_Register *RegisterSession) AddrToIdx(arg0 common.Address) (*big.Int, error) {
	return _Register.Contract.AddrToIdx(&_Register.CallOpts, arg0)
}

// AddrToIdx is a free data retrieval call binding the contract method 0x5f609695.
//
// Solidity: function addrToIdx(address ) view returns(uint256)
func (_Register *RegisterCallerSession) AddrToIdx(arg0 common.Address) (*big.Int, error) {
	return _Register.Contract.AddrToIdx(&_Register.CallOpts, arg0)
}

// VerifierMetaData contains all meta data concerning the Verifier contract.
var VerifierMetaData = &bind.MetaData{
	ABI: "[{\"constant\":true,\"inputs\":[],\"name\":\"description\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":false,\"inputs\":[{\"name\":\"\",\"type\":\"bytes\"}],\"name\":\"verify\",\"outputs\":[{\"name\":\"\",\"type\":\"bool\"}],\"payable\":false,\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Sigs: map[string]string{
		"7284e416": "description()",
		"8e760afe": "verify(bytes)",
	},
}

// VerifierABI is the input ABI used to generate the binding from.
// Deprecated: Use VerifierMetaData.ABI instead.
var VerifierABI = VerifierMetaData.ABI

// Deprecated: Use VerifierMetaData.Sigs instead.
// VerifierFuncSigs maps the 4-byte function signature to its string representation.
var VerifierFuncSigs = VerifierMetaData.Sigs

// Verifier is an auto generated Go binding around an Ethereum contract.
type Verifier struct {
	VerifierCaller     // Read-only binding to the contract
	VerifierTransactor // Write-only binding to the contract
	VerifierFilterer   // Log filterer for contract events
}

// VerifierCaller is an auto generated read-only Go binding around an Ethereum contract.
type VerifierCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// VerifierTransactor is an auto generated write-only Go binding around an Ethereum contract.
type VerifierTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// VerifierFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type VerifierFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// VerifierSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type VerifierSession struct {
	Contract     *Verifier         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// VerifierCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type VerifierCallerSession struct {
	Contract *VerifierCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// VerifierTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type VerifierTransactorSession struct {
	Contract     *VerifierTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// VerifierRaw is an auto generated low-level Go binding around an Ethereum contract.
type VerifierRaw struct {
	Contract *Verifier // Generic contract binding to access the raw methods on
}

// VerifierCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type VerifierCallerRaw struct {
	Contract *VerifierCaller // Generic read-only contract binding to access the raw methods on
}

// VerifierTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type VerifierTransactorRaw struct {
	Contract *VerifierTransactor // Generic write-only contract binding to access the raw methods on
}

// NewVerifier creates a new instance of Verifier, bound to a specific deployed contract.
func NewVerifier(address common.Address, backend bind.ContractBackend) (*Verifier, error) {
	contract, err := bindVerifier(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Verifier{VerifierCaller: VerifierCaller{contract: contract}, VerifierTransactor: VerifierTransactor{contract: contract}, VerifierFilterer: VerifierFilterer{contract: contract}}, nil
}

// NewVerifierCaller creates a new read-only instance of Verifier, bound to a specific deployed contract.
func NewVerifierCaller(address common.Address, caller bind.ContractCaller) (*VerifierCaller, error) {
	contract, err := bindVerifier(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &VerifierCaller{contract: contract}, nil
}

// NewVerifierTransactor creates a new write-only instance of Verifier, bound to a specific deployed contract.
func NewVerifierTransactor(address common.Address, transactor bind.ContractTransactor) (*VerifierTransactor, error) {
	contract, err := bindVerifier(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &VerifierTransactor{contract: contract}, nil
}

// NewVerifierFilterer creates a new log filterer instance of Verifier, bound to a specific deployed contract.
func NewVerifierFilterer(address common.Address, filterer bind.ContractFilterer) (*VerifierFilterer, error) {
	contract, err := bindVerifier(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &VerifierFilterer{contract: contract}, nil
}

// bindVerifier binds a generic wrapper to an already deployed contract.
func bindVerifier(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(VerifierABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Verifier *VerifierRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Verifier.Contract.VerifierCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Verifier *VerifierRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Verifier.Contract.VerifierTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Verifier *VerifierRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Verifier.Contract.VerifierTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Verifier *VerifierCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Verifier.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Verifier *VerifierTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Verifier.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Verifier *VerifierTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Verifier.Contract.contract.Transact(opts, method, params...)
}

// Description is a free data retrieval call binding the contract method 0x7284e416.
//
// Solidity: function description() view returns(string)
func (_Verifier *VerifierCaller) Description(opts *bind.CallOpts) (string, error) {
	var out []interface{}
	err := _Verifier.contract.Call(opts, &out, "description")

	if err != nil {
		return *new(string), err
	}

	out0 := *abi.ConvertType(out[0], new(string)).(*string)

	return out0, err

}

// Description is a free data retrieval call binding the contract method 0x7284e416.
//
// Solidity: function description() view returns(string)
func (_Verifier *VerifierSession) Description() (string, error) {
	return _Verifier.Contract.Description(&_Verifier.CallOpts)
}

// Description is a free data retrieval call binding the contract method 0x7284e416.
//
// Solidity: function description() view returns(string)
func (_Verifier *VerifierCallerSession) Description() (string, error) {
	return _Verifier.Contract.Description(&_Verifier.CallOpts)
}

// Verify is a paid mutator transaction binding the contract method 0x8e760afe.
//
// Solidity: function verify(bytes ) returns(bool)
func (_Verifier *VerifierTransactor) Verify(opts *bind.TransactOpts, arg0 []byte) (*types.Transaction, error) {
	return _Verifier.contract.Transact(opts, "verify", arg0)
}

// Verify is a paid mutator transaction binding the contract method 0x8e760afe.
//
// Solidity: function verify(bytes ) returns(bool)
func (_Verifier *VerifierSession) Verify(arg0 []byte) (*types.Transaction, error) {
	return _Verifier.Contract.Verify(&_Verifier.TransactOpts, arg0)
}

// Verify is a paid mutator transaction binding the contract method 0x8e760afe.
//
// Solidity: function verify(bytes ) returns(bool)
func (_Verifier *VerifierTransactorSession) Verify(arg0 []byte) (*types.Transaction, error) {
	return _Verifier.Contract.Verify(&_Verifier.TransactOpts, arg0)
}
