pragma solidity ^0.5.0;

contract SimpleStorage {
    uint pos0;
    uint pos2;
    mapping(address => uint) pos1;
    constructor() public{
        pos0 = 12345;
        pos1[msg.sender] = 5678;
        pos2 = 1234;
    }
}