pragma solidity ^0.4.24;

import "openzeppelin-solidity/contracts/token/ERC20/ERC20.sol";

/**
 * @title ERC20 interface for token contracts deployed to Loom DAppChains.
 */
contract ERC20DAppToken is ERC20 {
    // Called by the DAppChain DPOS contract to mint tokens
    //
    // NOTE: This function will only be called by the DAppChain DPOS contract to mint ERC20 tokens
    function mintToDPOS(uint256 amount, address _dposAddress) public;
}
