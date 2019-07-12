pragma solidity ^0.4.28;

import "openzeppelin-solidity/contracts/token/ERC20/StandardToken.sol";
import "./ERC20DAppToken.sol";

contract SampleERC20Token is ERC20DAppToken, StandardToken {
    // DPOS contract address
    address public dpos;

    string public name = "ERC20";
    string public symbol = "ERC";
    uint8 public decimals = 6;
    
    /**
     * @dev Constructor function
     */
    constructor(address _dpos) public {
        dpos = _dpos;
        totalSupply_ = 1000000000 * (10 ** uint256(decimals));
        balances[_dpos] = totalSupply_;
    }

    function mintToDPOS(uint256 _amount) public {
        require(msg.sender == gateway);
        totalSupply_ = totalSupply_.add(_amount);
        balances[dpos] = balances[dpos].add(_amount);
    }
}
