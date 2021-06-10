pragma solidity >=0.4.21;

import "./LoomNativeApi.sol";

library TestLoomNativeApi {
    function TestMappedLoomAccount(string memory fromChain, bytes32 hash, uint8 v, bytes32 r, bytes32 s) public view returns (address) {
        return LoomNativeApi.MappedAccount(fromChain, ecrecover(hash, v, r, s));
    }

    function TestMappedAccount(string memory fromChain, bytes32 hash, uint8 v, bytes32 r, bytes32 s, string memory toChain) public view returns (address) {
        return LoomNativeApi.MappedAccount(fromChain, ecrecover(hash, v, r, s), toChain);
    }
}
