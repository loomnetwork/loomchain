pragma solidity ^0.5.0;

contract EventTestContract {
    uint256 public value;
  
    event NewValueSet(uint256 indexed _value);
    event AnotherValueSet(uint256 indexed _value);
  
    function set(uint256 _value) public {
      value = _value;
      emit NewValueSet(value);
      emit NewValueSet(value + 1);
      emit AnotherValueSet(value + 1);
    }
  }