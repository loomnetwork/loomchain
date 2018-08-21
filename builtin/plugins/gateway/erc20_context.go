package gateway

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	loom "github.com/loomnetwork/go-loom"
	contract "github.com/loomnetwork/go-loom/plugin/contractpb"
)

// Helper for making calls into an ERC20 contract in the Loom EVM.
type erc20StaticContext struct {
	ctx contract.StaticContext
	// Address of ERC20 contract deployed to Loom EVM.
	tokenAddr   loom.Address
	contractABI *abi.ABI
}

func newERC20StaticContext(ctx contract.StaticContext, tokenAddr loom.Address) *erc20StaticContext {
	erc20ABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		panic(err)
	}
	return &erc20StaticContext{
		ctx:         ctx,
		tokenAddr:   tokenAddr,
		contractABI: &erc20ABI,
	}
}

func (c *erc20StaticContext) balanceOf(owner loom.Address) (*big.Int, error) {
	ownerAddr := common.BytesToAddress(owner.Local)
	var result *big.Int
	if err := c.staticCallEVM("balanceOf", &result, ownerAddr); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *erc20StaticContext) staticCallEVM(method string, result interface{}, params ...interface{}) error {
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

// Helper for making calls into an ERC20 contract in the Loom EVM.
type erc20Context struct {
	*erc20StaticContext
	ctx contract.Context
}

func newERC20Context(ctx contract.Context, tokenAddr loom.Address) *erc20Context {
	return &erc20Context{
		erc20StaticContext: newERC20StaticContext(ctx, tokenAddr),
		ctx:                ctx,
	}
}

func (c *erc20Context) transferFrom(from, to loom.Address, amount *big.Int) error {
	fromAddr := common.BytesToAddress(from.Local)
	toAddr := common.BytesToAddress(to.Local)
	_, err := c.callEVM("transferFrom", fromAddr, toAddr, amount)
	return err
}

func (c *erc20Context) transfer(to loom.Address, amount *big.Int) error {
	toAddr := common.BytesToAddress(to.Local)
	_, err := c.callEVM("transfer", toAddr, amount)
	return err
}

func (c *erc20Context) mintToGateway(amount *big.Int) error {
	_, err := c.callEVM("mintToGateway", amount)
	return err
}

func (c *erc20Context) callEVM(method string, params ...interface{}) ([]byte, error) {
	input, err := c.contractABI.Pack(method, params...)
	if err != nil {
		return nil, err
	}
	var evmOut []byte
	return evmOut, contract.CallEVM(c.ctx, c.tokenAddr, input, &evmOut)
}

// From src/ethcontract/ERC20DAppToken.abi in transfer-gateway-v2 repo
const erc20ABI = `
[
	{
	  "constant": false,
	  "inputs": [
		{
		  "name": "spender",
		  "type": "address"
		},
		{
		  "name": "value",
		  "type": "uint256"
		}
	  ],
	  "name": "approve",
	  "outputs": [
		{
		  "name": "",
		  "type": "bool"
		}
	  ],
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
		  "name": "from",
		  "type": "address"
		},
		{
		  "name": "to",
		  "type": "address"
		},
		{
		  "name": "value",
		  "type": "uint256"
		}
	  ],
	  "name": "transferFrom",
	  "outputs": [
		{
		  "name": "",
		  "type": "bool"
		}
	  ],
	  "payable": false,
	  "stateMutability": "nonpayable",
	  "type": "function"
	},
	{
	  "constant": true,
	  "inputs": [
		{
		  "name": "who",
		  "type": "address"
		}
	  ],
	  "name": "balanceOf",
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
		  "name": "to",
		  "type": "address"
		},
		{
		  "name": "value",
		  "type": "uint256"
		}
	  ],
	  "name": "transfer",
	  "outputs": [
		{
		  "name": "",
		  "type": "bool"
		}
	  ],
	  "payable": false,
	  "stateMutability": "nonpayable",
	  "type": "function"
	},
	{
	  "constant": true,
	  "inputs": [
		{
		  "name": "owner",
		  "type": "address"
		},
		{
		  "name": "spender",
		  "type": "address"
		}
	  ],
	  "name": "allowance",
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
	  "anonymous": false,
	  "inputs": [
		{
		  "indexed": true,
		  "name": "owner",
		  "type": "address"
		},
		{
		  "indexed": true,
		  "name": "spender",
		  "type": "address"
		},
		{
		  "indexed": false,
		  "name": "value",
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
		  "name": "from",
		  "type": "address"
		},
		{
		  "indexed": true,
		  "name": "to",
		  "type": "address"
		},
		{
		  "indexed": false,
		  "name": "value",
		  "type": "uint256"
		}
	  ],
	  "name": "Transfer",
	  "type": "event"
	},
	{
	  "constant": false,
	  "inputs": [
		{
		  "name": "amount",
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
