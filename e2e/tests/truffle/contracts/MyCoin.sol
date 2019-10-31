pragma solidity ^0.5.0;

import "openzeppelin-solidity/contracts/token/ERC20/ERC20.sol";

contract MyCoin is ERC20 {
    string public name = "MyCoin";
    string public symbol = "MCC";
    uint8 public decimals = 18;

    // one billion in initial supply
    uint256 public constant INITIAL_SUPPLY = 1000000000;

    bytes4 constant ERC20_RECEIVED = 0xbc04f0af;
    uint setFee = 0;
    uint globalUint = 0;
    constructor() public {
        _mint(msg.sender, INITIAL_SUPPLY * (10 ** uint256(decimals)));
    }

    function easyTransferTo(address recipient,uint256 amount) public {
        _transfer(msg.sender,recipient,amount);
    }

    function approveForTransfer(address recipient,uint256 amount) public returns (bool) {
        _approve(msg.sender,recipient,amount);
        return true;
    }

    function transferFrom(address sender, address recipient, uint256 amount) public returns (bool) {
        _transfer(sender, recipient, amount);
        return true;
    }

    event NewValueSet(uint _value);

    function payToSet(uint num) external payable{
        require(msg.value >= setFee,"value sent must greater than or equal to setFee");
        globalUint = num;
        emit NewValueSet(num);
    }

    function getUint() public view returns (uint){
        return globalUint;
    }

}
