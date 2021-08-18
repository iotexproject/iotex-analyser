package main

const (
	// AirdripABI defines the ABI of Airdrip contract
	AirdripABI = `[
		{
		  "inputs": [
			{
			  "internalType": "address",
			  "name": "_token",
			  "type": "address"
			},
			{
			  "internalType": "uint256",
			  "name": "_exchangeRate",
			  "type": "uint256"
			},
			{
			  "internalType": "uint256",
			  "name": "_fadeoutDuration",
			  "type": "uint256"
			},
			{
			  "internalType": "uint256",
			  "name": "_maxDuration",
			  "type": "uint256"
			}
		  ],
		  "payable": false,
		  "stateMutability": "nonpayable",
		  "type": "constructor"
		},
		{
		  "anonymous": false,
		  "inputs": [
			{
			  "indexed": true,
			  "internalType": "address",
			  "name": "user",
			  "type": "address"
			},
			{
			  "indexed": false,
			  "internalType": "uint256",
			  "name": "points",
			  "type": "uint256"
			}
		  ],
		  "name": "Claimed",
		  "type": "event"
		},
		{
		  "anonymous": false,
		  "inputs": [
			{
			  "indexed": true,
			  "internalType": "address",
			  "name": "asset",
			  "type": "address"
			},
			{
			  "indexed": false,
			  "internalType": "uint256",
			  "name": "volume",
			  "type": "uint256"
			}
		  ],
		  "name": "Dripped",
		  "type": "event"
		},
		{
		  "anonymous": false,
		  "inputs": [
			{
			  "indexed": false,
			  "internalType": "uint256",
			  "name": "rate",
			  "type": "uint256"
			}
		  ],
		  "name": "ExchangeRateUpdated",
		  "type": "event"
		},
		{
		  "anonymous": false,
		  "inputs": [
			{
			  "indexed": true,
			  "internalType": "address",
			  "name": "user",
			  "type": "address"
			},
			{
			  "indexed": false,
			  "internalType": "uint256",
			  "name": "amount",
			  "type": "uint256"
			},
			{
			  "indexed": false,
			  "internalType": "uint256",
			  "name": "exchangeRate",
			  "type": "uint256"
			}
		  ],
		  "name": "Exchanged",
		  "type": "event"
		},
		{
		  "anonymous": false,
		  "inputs": [
			{
			  "indexed": true,
			  "internalType": "address",
			  "name": "user",
			  "type": "address"
			}
		  ],
		  "name": "NewRegistration",
		  "type": "event"
		},
		{
		  "anonymous": false,
		  "inputs": [
			{
			  "indexed": true,
			  "internalType": "uint256",
			  "name": "value",
			  "type": "uint256"
			}
		  ],
		  "name": "NewTerm",
		  "type": "event"
		},
		{
		  "anonymous": false,
		  "inputs": [
			{
			  "indexed": true,
			  "internalType": "address",
			  "name": "previousOwner",
			  "type": "address"
			},
			{
			  "indexed": true,
			  "internalType": "address",
			  "name": "newOwner",
			  "type": "address"
			}
		  ],
		  "name": "OwnershipTransferred",
		  "type": "event"
		},
		{
		  "anonymous": false,
		  "inputs": [
			{
			  "indexed": true,
			  "internalType": "address",
			  "name": "provider",
			  "type": "address"
			},
			{
			  "indexed": true,
			  "internalType": "address",
			  "name": "asset",
			  "type": "address"
			},
			{
			  "indexed": false,
			  "internalType": "uint256",
			  "name": "amount",
			  "type": "uint256"
			},
			{
			  "indexed": false,
			  "internalType": "uint256",
			  "name": "endBlock",
			  "type": "uint256"
			},
			{
			  "indexed": false,
			  "internalType": "uint256",
			  "name": "konstante",
			  "type": "uint256"
			}
		  ],
		  "name": "Poured",
		  "type": "event"
		},
		{
		  "anonymous": false,
		  "inputs": [
			{
			  "indexed": true,
			  "internalType": "address",
			  "name": "user",
			  "type": "address"
			},
			{
			  "indexed": true,
			  "internalType": "address",
			  "name": "asset",
			  "type": "address"
			},
			{
			  "indexed": false,
			  "internalType": "uint256",
			  "name": "amount",
			  "type": "uint256"
			},
			{
			  "indexed": false,
			  "internalType": "uint256",
			  "name": "points",
			  "type": "uint256"
			}
		  ],
		  "name": "Redeemed",
		  "type": "event"
		},
		{
		  "constant": true,
		  "inputs": [
			{
			  "internalType": "address",
			  "name": "",
			  "type": "address"
			}
		  ],
		  "name": "accounts",
		  "outputs": [
			{
			  "internalType": "uint256",
			  "name": "shares",
			  "type": "uint256"
			},
			{
			  "internalType": "uint256",
			  "name": "debt",
			  "type": "uint256"
			},
			{
			  "internalType": "uint256",
			  "name": "balance",
			  "type": "uint256"
			},
			{
			  "internalType": "uint256",
			  "name": "since",
			  "type": "uint256"
			}
		  ],
		  "payable": false,
		  "stateMutability": "view",
		  "type": "function"
		},
		{
		  "constant": true,
		  "inputs": [
			{
			  "internalType": "uint256",
			  "name": "",
			  "type": "uint256"
			}
		  ],
		  "name": "assets",
		  "outputs": [
			{
			  "internalType": "address",
			  "name": "",
			  "type": "address"
			}
		  ],
		  "payable": false,
		  "stateMutability": "view",
		  "type": "function"
		},
		{
		  "constant": true,
		  "inputs": [],
		  "name": "exchangeRate",
		  "outputs": [
			{
			  "internalType": "uint256",
			  "name": "",
			  "type": "uint256"
			}
		  ],
		  "payable": false,
		  "stateMutability": "view",
		  "type": "function"
		},
		{
		  "constant": true,
		  "inputs": [],
		  "name": "fadeoutDuration",
		  "outputs": [
			{
			  "internalType": "uint256",
			  "name": "",
			  "type": "uint256"
			}
		  ],
		  "payable": false,
		  "stateMutability": "view",
		  "type": "function"
		},
		{
		  "constant": true,
		  "inputs": [],
		  "name": "isOwner",
		  "outputs": [
			{
			  "internalType": "bool",
			  "name": "",
			  "type": "bool"
			}
		  ],
		  "payable": false,
		  "stateMutability": "view",
		  "type": "function"
		},
		{
		  "constant": true,
		  "inputs": [],
		  "name": "maxDuration",
		  "outputs": [
			{
			  "internalType": "uint256",
			  "name": "",
			  "type": "uint256"
			}
		  ],
		  "payable": false,
		  "stateMutability": "view",
		  "type": "function"
		},
		{
		  "constant": true,
		  "inputs": [],
		  "name": "owner",
		  "outputs": [
			{
			  "internalType": "address",
			  "name": "",
			  "type": "address"
			}
		  ],
		  "payable": false,
		  "stateMutability": "view",
		  "type": "function"
		},
		{
		  "constant": true,
		  "inputs": [
			{
			  "internalType": "address",
			  "name": "",
			  "type": "address"
			}
		  ],
		  "name": "pools",
		  "outputs": [
			{
			  "internalType": "uint256",
			  "name": "konstante",
			  "type": "uint256"
			},
			{
			  "internalType": "uint256",
			  "name": "startBlock",
			  "type": "uint256"
			},
			{
			  "internalType": "uint256",
			  "name": "endBlock",
			  "type": "uint256"
			},
			{
			  "internalType": "uint256",
			  "name": "lastDripBlock",
			  "type": "uint256"
			},
			{
			  "internalType": "uint256",
			  "name": "volume",
			  "type": "uint256"
			},
			{
			  "internalType": "uint256",
			  "name": "cloudVolume",
			  "type": "uint256"
			},
			{
			  "internalType": "bool",
			  "name": "exists",
			  "type": "bool"
			}
		  ],
		  "payable": false,
		  "stateMutability": "view",
		  "type": "function"
		},
		{
		  "constant": false,
		  "inputs": [],
		  "name": "renounceOwnership",
		  "outputs": [],
		  "payable": false,
		  "stateMutability": "nonpayable",
		  "type": "function"
		},
		{
		  "constant": true,
		  "inputs": [],
		  "name": "term",
		  "outputs": [
			{
			  "internalType": "uint256",
			  "name": "",
			  "type": "uint256"
			}
		  ],
		  "payable": false,
		  "stateMutability": "view",
		  "type": "function"
		},
		{
		  "constant": true,
		  "inputs": [],
		  "name": "token",
		  "outputs": [
			{
			  "internalType": "contract IBurnable",
			  "name": "",
			  "type": "address"
			}
		  ],
		  "payable": false,
		  "stateMutability": "view",
		  "type": "function"
		},
		{
		  "constant": false,
		  "inputs": [
			{
			  "internalType": "address",
			  "name": "newOwner",
			  "type": "address"
			}
		  ],
		  "name": "transferOwnership",
		  "outputs": [],
		  "payable": false,
		  "stateMutability": "nonpayable",
		  "type": "function"
		},
		{
		  "constant": true,
		  "inputs": [],
		  "name": "volumeScale",
		  "outputs": [
			{
			  "internalType": "uint256",
			  "name": "",
			  "type": "uint256"
			}
		  ],
		  "payable": false,
		  "stateMutability": "view",
		  "type": "function"
		},
		{
		  "constant": false,
		  "inputs": [],
		  "name": "register",
		  "outputs": [],
		  "payable": false,
		  "stateMutability": "nonpayable",
		  "type": "function"
		},
		{
		  "constant": true,
		  "inputs": [
			{
			  "internalType": "address",
			  "name": "_user",
			  "type": "address"
			}
		  ],
		  "name": "balanceOf",
		  "outputs": [
			{
			  "internalType": "uint256",
			  "name": "",
			  "type": "uint256"
			}
		  ],
		  "payable": false,
		  "stateMutability": "view",
		  "type": "function"
		},
		{
		  "constant": false,
		  "inputs": [
			{
			  "internalType": "address",
			  "name": "_user",
			  "type": "address"
			}
		  ],
		  "name": "claim",
		  "outputs": [],
		  "payable": false,
		  "stateMutability": "nonpayable",
		  "type": "function"
		},
		{
		  "constant": false,
		  "inputs": [
			{
			  "internalType": "uint256",
			  "name": "_amount",
			  "type": "uint256"
			}
		  ],
		  "name": "exchange",
		  "outputs": [],
		  "payable": false,
		  "stateMutability": "nonpayable",
		  "type": "function"
		},
		{
		  "constant": false,
		  "inputs": [
			{
			  "internalType": "uint256",
			  "name": "_newRate",
			  "type": "uint256"
			}
		  ],
		  "name": "setExchangeRate",
		  "outputs": [],
		  "payable": false,
		  "stateMutability": "nonpayable",
		  "type": "function"
		},
		{
		  "constant": false,
		  "inputs": [
			{
			  "internalType": "address[]",
			  "name": "_users",
			  "type": "address[]"
			},
			{
			  "internalType": "uint256[]",
			  "name": "_shares",
			  "type": "uint256[]"
			}
		  ],
		  "name": "updateShares",
		  "outputs": [],
		  "payable": false,
		  "stateMutability": "nonpayable",
		  "type": "function"
		},
		{
		  "constant": false,
		  "inputs": [],
		  "name": "nextTerm",
		  "outputs": [],
		  "payable": false,
		  "stateMutability": "nonpayable",
		  "type": "function"
		},
		{
		  "constant": true,
		  "inputs": [
			{
			  "internalType": "address",
			  "name": "_asset",
			  "type": "address"
			}
		  ],
		  "name": "poolOf",
		  "outputs": [
			{
			  "components": [
				{
				  "internalType": "uint256",
				  "name": "konstante",
				  "type": "uint256"
				},
				{
				  "internalType": "uint256",
				  "name": "startBlock",
				  "type": "uint256"
				},
				{
				  "internalType": "uint256",
				  "name": "endBlock",
				  "type": "uint256"
				},
				{
				  "internalType": "uint256",
				  "name": "lastDripBlock",
				  "type": "uint256"
				},
				{
				  "internalType": "uint256",
				  "name": "volume",
				  "type": "uint256"
				},
				{
				  "internalType": "uint256",
				  "name": "cloudVolume",
				  "type": "uint256"
				},
				{
				  "internalType": "bool",
				  "name": "exists",
				  "type": "bool"
				}
			  ],
			  "internalType": "struct Airdrip.AssetPool",
			  "name": "",
			  "type": "tuple"
			}
		  ],
		  "payable": false,
		  "stateMutability": "view",
		  "type": "function"
		},
		{
		  "constant": false,
		  "inputs": [
			{
			  "internalType": "address",
			  "name": "_asset",
			  "type": "address"
			}
		  ],
		  "name": "drip",
		  "outputs": [],
		  "payable": false,
		  "stateMutability": "nonpayable",
		  "type": "function"
		},
		{
		  "constant": false,
		  "inputs": [
			{
			  "internalType": "address",
			  "name": "_asset",
			  "type": "address"
			},
			{
			  "internalType": "uint256",
			  "name": "_endBlock",
			  "type": "uint256"
			}
		  ],
		  "name": "extendDuration",
		  "outputs": [],
		  "payable": false,
		  "stateMutability": "nonpayable",
		  "type": "function"
		},
		{
		  "constant": false,
		  "inputs": [
			{
			  "internalType": "address",
			  "name": "_asset",
			  "type": "address"
			},
			{
			  "internalType": "uint256",
			  "name": "_amount",
			  "type": "uint256"
			}
		  ],
		  "name": "increaseSupply",
		  "outputs": [],
		  "payable": false,
		  "stateMutability": "nonpayable",
		  "type": "function"
		},
		{
		  "constant": false,
		  "inputs": [
			{
			  "internalType": "address",
			  "name": "_asset",
			  "type": "address"
			},
			{
			  "internalType": "uint256",
			  "name": "_amount",
			  "type": "uint256"
			},
			{
			  "internalType": "uint256",
			  "name": "_duration",
			  "type": "uint256"
			},
			{
			  "internalType": "uint256",
			  "name": "_konstante",
			  "type": "uint256"
			}
		  ],
		  "name": "addAsset",
		  "outputs": [],
		  "payable": false,
		  "stateMutability": "nonpayable",
		  "type": "function"
		},
		{
		  "constant": true,
		  "inputs": [
			{
			  "internalType": "address",
			  "name": "_asset",
			  "type": "address"
			}
		  ],
		  "name": "volumeToDripPerBlock",
		  "outputs": [
			{
			  "internalType": "uint256",
			  "name": "",
			  "type": "uint256"
			}
		  ],
		  "payable": false,
		  "stateMutability": "view",
		  "type": "function"
		},
		{
		  "constant": true,
		  "inputs": [],
		  "name": "numOfAssets",
		  "outputs": [
			{
			  "internalType": "uint256",
			  "name": "",
			  "type": "uint256"
			}
		  ],
		  "payable": false,
		  "stateMutability": "view",
		  "type": "function"
		},
		{
		  "constant": true,
		  "inputs": [
			{
			  "internalType": "address",
			  "name": "_asset",
			  "type": "address"
			},
			{
			  "internalType": "uint256",
			  "name": "_amount",
			  "type": "uint256"
			}
		  ],
		  "name": "redeemExactAssetCost",
		  "outputs": [
			{
			  "internalType": "uint256",
			  "name": "",
			  "type": "uint256"
			}
		  ],
		  "payable": false,
		  "stateMutability": "view",
		  "type": "function"
		},
		{
		  "constant": false,
		  "inputs": [
			{
			  "internalType": "address",
			  "name": "_asset",
			  "type": "address"
			},
			{
			  "internalType": "uint256",
			  "name": "_amount",
			  "type": "uint256"
			},
			{
			  "internalType": "uint256",
			  "name": "_maxPoints",
			  "type": "uint256"
			}
		  ],
		  "name": "redeemExactAsset",
		  "outputs": [],
		  "payable": false,
		  "stateMutability": "nonpayable",
		  "type": "function"
		},
		{
		  "constant": true,
		  "inputs": [
			{
			  "internalType": "address",
			  "name": "_asset",
			  "type": "address"
			},
			{
			  "internalType": "uint256",
			  "name": "_points",
			  "type": "uint256"
			}
		  ],
		  "name": "redeemAmount",
		  "outputs": [
			{
			  "internalType": "uint256",
			  "name": "",
			  "type": "uint256"
			}
		  ],
		  "payable": false,
		  "stateMutability": "view",
		  "type": "function"
		},
		{
		  "constant": false,
		  "inputs": [
			{
			  "internalType": "address",
			  "name": "_asset",
			  "type": "address"
			},
			{
			  "internalType": "uint256",
			  "name": "_minAmount",
			  "type": "uint256"
			},
			{
			  "internalType": "uint256",
			  "name": "_points",
			  "type": "uint256"
			}
		  ],
		  "name": "redeem",
		  "outputs": [],
		  "payable": false,
		  "stateMutability": "nonpayable",
		  "type": "function"
		}
	  ]`
)
