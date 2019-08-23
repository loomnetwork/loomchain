pragma solidity ^0.5.0;

contract NonceTestContract {
  uint value;

  event NewValueSet(uint _value);

  function set(uint _value) public {
    value = _value;
    emit NewValueSet(value);
  }

  function get() public view returns (uint) {
    return value;
  }

  function err() public {
    revert("Revert");
  }
}
