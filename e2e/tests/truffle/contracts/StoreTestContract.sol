pragma solidity ^0.5.0;

contract StoreTestContract {
    uint storedUint1 = 15;
    uint128 storedUint2 = 17055;
    uint32 storedUint3 = 1234;

    bytes16 string1 = "test1";
    bytes32 string2 = "test1236";
    bytes32 string3 = "lets string something";

    uint[] uintarray;

    constructor() public {
        uintarray.push(8000);
        uintarray.push(9000);
    }
}
