pragma solidity ^0.4.24;

import 'zeppelin-solidity/contracts/token/ERC20/MintableToken.sol';

contract SampleERC20Token is MintableToken{
    address public dpos;
    string public name = "ERC20";
    string public symbol = "ERC";
    uint8 public constant decimals = 6;
    uint256 public INITIAL_SUPPLY = 10000 * (10 ** uint256(decimals));
    event mintTODPOS(uint256 _amount, address dpos);

    constructor(address _dpos) public {
        dpos = _dpos;
        totalSupply_ = INITIAL_SUPPLY;
    }

    function mintToDPOS(uint256 _amount) public {
        require(msg.sender == dpos, "only the DPOS is allowed to mint");
        totalSupply_ = totalSupply_.add(_amount);
        balances[dpos] = balances[dpos].add(_amount);
        emit mintTODPOS(_amount,dpos);
    }

}
