package gateway

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	loom "github.com/loomnetwork/go-loom"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
)

type erc721StaticContext struct {
	ctx contract.StaticContext
	// Address of ERC721 contract deployed to Loom EVM.
	tokenAddr   loom.Address
	contractABI *abi.ABI
}

func newERC721StaticContext(ctx contract.StaticContext, tokenAddr loom.Address) *erc721StaticContext {
	erc721ABI, err := abi.JSON(strings.NewReader(erc721ABI))
	if err != nil {
		panic(err)
	}
	return &erc721StaticContext{
		ctx:         ctx,
		tokenAddr:   tokenAddr,
		contractABI: &erc721ABI,
	}
}

func (c *erc721StaticContext) exists(tokenID *big.Int) (bool, error) {
	var result bool
	return result, c.staticCallEVM("exists", &result, tokenID)
}

func (c *erc721StaticContext) ownerOf(tokenID *big.Int) (loom.Address, error) {
	var result common.Address
	if err := c.staticCallEVM("ownerOf", &result, tokenID); err != nil {
		return loom.Address{}, err
	}
	return loom.Address{
		ChainID: c.ctx.Block().ChainID,
		Local:   result.Bytes(),
	}, nil
}

func (c *erc721StaticContext) staticCallEVM(method string, result interface{}, params ...interface{}) error {
	input, err := c.contractABI.Pack(method, params...)
	if err != nil {
		return err
	}
	var output []byte
	if err := contract.StaticCallEVM(c.ctx, c.tokenAddr, input, &output); err != nil {
		return err
	}
	return c.contractABI.Unpack(result, method, output)
}

type erc721Context struct {
	*erc721StaticContext
	ctx contract.Context
}

func newERC721Context(ctx contract.Context, tokenAddr loom.Address) *erc721Context {
	return &erc721Context{
		erc721StaticContext: newERC721StaticContext(ctx, tokenAddr),
		ctx:                 ctx,
	}
}

func (c *erc721Context) mintToGateway(tokenID *big.Int) error {
	_, err := c.callEVM("mintToGateway", tokenID)
	return err
}

func (c *erc721Context) safeTransferFrom(from, to loom.Address, tokenID *big.Int) error {
	fromAddr := common.BytesToAddress(from.Local)
	toAddr := common.BytesToAddress(to.Local)
	_, err := c.callEVM("safeTransferFrom", fromAddr, toAddr, tokenID, []byte{})
	return err
}

func (c *erc721Context) approve(spender loom.Address, tokenID *big.Int) error {
	spenderAddr := common.BytesToAddress(spender.Local)
	_, err := c.callEVM("approve", spenderAddr, tokenID)
	return err
}

func (c *erc721Context) callEVM(method string, params ...interface{}) ([]byte, error) {
	input, err := c.contractABI.Pack(method, params...)
	if err != nil {
		return nil, err
	}
	var evmOut []byte
	return evmOut, contract.CallEVM(c.ctx, c.tokenAddr, input, &evmOut)
}

// From src/ethcontract/ERC721DAppToken.abi in transfer-gateway-v2 repo
const erc721ABI = `
[
	{
	  "constant": true,
	  "inputs": [],
	  "name": "name",
	  "outputs": [
		{
		  "name": "_name",
		  "type": "string"
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
		  "name": "_tokenId",
		  "type": "uint256"
		}
	  ],
	  "name": "getApproved",
	  "outputs": [
		{
		  "name": "_operator",
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
		  "name": "_to",
		  "type": "address"
		},
		{
		  "name": "_tokenId",
		  "type": "uint256"
		}
	  ],
	  "name": "approve",
	  "outputs": [],
	  "payable": false,
	  "stateMutability": "nonpayable",
	  "type": "function"
	},
	{
	  "constant": true,
	  "inputs": [],
	  "name": "totalSupply",
	  "outputs": [
		{
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
		  "name": "_from",
		  "type": "address"
		},
		{
		  "name": "_to",
		  "type": "address"
		},
		{
		  "name": "_tokenId",
		  "type": "uint256"
		}
	  ],
	  "name": "transferFrom",
	  "outputs": [],
	  "payable": false,
	  "stateMutability": "nonpayable",
	  "type": "function"
	},
	{
	  "constant": true,
	  "inputs": [
		{
		  "name": "_owner",
		  "type": "address"
		},
		{
		  "name": "_index",
		  "type": "uint256"
		}
	  ],
	  "name": "tokenOfOwnerByIndex",
	  "outputs": [
		{
		  "name": "_tokenId",
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
		  "name": "_from",
		  "type": "address"
		},
		{
		  "name": "_to",
		  "type": "address"
		},
		{
		  "name": "_tokenId",
		  "type": "uint256"
		}
	  ],
	  "name": "safeTransferFrom",
	  "outputs": [],
	  "payable": false,
	  "stateMutability": "nonpayable",
	  "type": "function"
	},
	{
	  "constant": true,
	  "inputs": [
		{
		  "name": "_tokenId",
		  "type": "uint256"
		}
	  ],
	  "name": "exists",
	  "outputs": [
		{
		  "name": "_exists",
		  "type": "bool"
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
		  "name": "_index",
		  "type": "uint256"
		}
	  ],
	  "name": "tokenByIndex",
	  "outputs": [
		{
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
		  "name": "_tokenId",
		  "type": "uint256"
		}
	  ],
	  "name": "ownerOf",
	  "outputs": [
		{
		  "name": "_owner",
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
		  "name": "_owner",
		  "type": "address"
		}
	  ],
	  "name": "balanceOf",
	  "outputs": [
		{
		  "name": "_balance",
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
	  "name": "symbol",
	  "outputs": [
		{
		  "name": "_symbol",
		  "type": "string"
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
		  "name": "_operator",
		  "type": "address"
		},
		{
		  "name": "_approved",
		  "type": "bool"
		}
	  ],
	  "name": "setApprovalForAll",
	  "outputs": [],
	  "payable": false,
	  "stateMutability": "nonpayable",
	  "type": "function"
	},
	{
	  "constant": false,
	  "inputs": [
		{
		  "name": "_from",
		  "type": "address"
		},
		{
		  "name": "_to",
		  "type": "address"
		},
		{
		  "name": "_tokenId",
		  "type": "uint256"
		},
		{
		  "name": "_data",
		  "type": "bytes"
		}
	  ],
	  "name": "safeTransferFrom",
	  "outputs": [],
	  "payable": false,
	  "stateMutability": "nonpayable",
	  "type": "function"
	},
	{
	  "constant": true,
	  "inputs": [
		{
		  "name": "_tokenId",
		  "type": "uint256"
		}
	  ],
	  "name": "tokenURI",
	  "outputs": [
		{
		  "name": "",
		  "type": "string"
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
		  "name": "_owner",
		  "type": "address"
		},
		{
		  "name": "_operator",
		  "type": "address"
		}
	  ],
	  "name": "isApprovedForAll",
	  "outputs": [
		{
		  "name": "",
		  "type": "bool"
		}
	  ],
	  "payable": false,
	  "stateMutability": "view",
	  "type": "function"
	},
	{
	  "anonymous": false,
	  "inputs": [
		{
		  "indexed": true,
		  "name": "_from",
		  "type": "address"
		},
		{
		  "indexed": true,
		  "name": "_to",
		  "type": "address"
		},
		{
		  "indexed": false,
		  "name": "_tokenId",
		  "type": "uint256"
		}
	  ],
	  "name": "Transfer",
	  "type": "event"
	},
	{
	  "anonymous": false,
	  "inputs": [
		{
		  "indexed": true,
		  "name": "_owner",
		  "type": "address"
		},
		{
		  "indexed": true,
		  "name": "_approved",
		  "type": "address"
		},
		{
		  "indexed": false,
		  "name": "_tokenId",
		  "type": "uint256"
		}
	  ],
	  "name": "Approval",
	  "type": "event"
	},
	{
	  "anonymous": false,
	  "inputs": [
		{
		  "indexed": true,
		  "name": "_owner",
		  "type": "address"
		},
		{
		  "indexed": true,
		  "name": "_operator",
		  "type": "address"
		},
		{
		  "indexed": false,
		  "name": "_approved",
		  "type": "bool"
		}
	  ],
	  "name": "ApprovalForAll",
	  "type": "event"
	},
	{
	  "constant": false,
	  "inputs": [
		{
		  "name": "_uid",
		  "type": "uint256"
		}
	  ],
	  "name": "mintToGateway",
	  "outputs": [],
	  "payable": false,
	  "stateMutability": "nonpayable",
	  "type": "function"
	}
]`
