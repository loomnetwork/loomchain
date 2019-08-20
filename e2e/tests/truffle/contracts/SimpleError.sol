pragma solidity ^0.5.0;

contract SimpleError {
  function err() public {
    revert("Revert");
  }
}