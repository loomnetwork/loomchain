pragma solidity ^0.4.24;

import "openzeppelin-solidity/contracts/token/ERC20/StandardToken.sol";
import "./ERC20DAppToken.sol";

contract SampleERC20Token is ERC20DAppToken, StandardToken {
    // DPOS contract address
    address public dpos;

    string public name = "ERC20";
    string public symbol = "ERC";
    uint8 public decimals = 6;
    event startMining();
    event mintingDPOS(uint256 _amount, address _dposAddress);

    /**
     * @dev Constructor function
     */
    constructor() public {
        emit startMining();
        totalSupply_ = 1000000000 * (10 ** uint256(decimals));
    }

    function mintToDPOS(uint256 _amount, address _dposAddress) public returns (bool ok){
        emit mintingDPOS(_amount,_dposAddress);
        totalSupply_ = totalSupply_.add(_amount);
        balances[_dposAddress] = balances[_dposAddress].add(_amount);
        return true;
    }
}
