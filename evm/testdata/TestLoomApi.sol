pragma solidity >=0.4.21;

import "./LoomApi.sol";

contract TestLoomApi {
    function TestMappedAccount(bytes32 hash, uint8 v, bytes32 r, bytes32 s) public view returns (address) {
        return LoomApi.MappedAccount("eth", ecrecover(hash, v, r, s));
    }
}
