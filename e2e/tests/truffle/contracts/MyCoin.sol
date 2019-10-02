pragma solidity ^0.5.0;

import "openzeppelin-solidity/contracts/token/ERC20/ERC20Mintable.sol";

contract MyCoin is ERC20Mintable {
    string public name = "MyCoin";
    string public symbol = "MCC";
    uint8 public decimals = 18;

    // one billion in initial supply
    uint256 public constant INITIAL_SUPPLY = 1000000000;

    bytes4 constant ERC20_RECEIVED = 0xbc04f0af;
    
    constructor() public {
        _mint(msg.sender, INITIAL_SUPPLY * (10 ** uint256(decimals)));
    }
}
