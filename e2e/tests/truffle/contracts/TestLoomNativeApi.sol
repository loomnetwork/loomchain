pragma solidity >=0.4.21;

import "./LoomNativeApi.sol";

contract TestLoomNativeApi {
    function TestMappedAccount(string memory ethChainId, bytes32 hash, uint8 v, bytes32 r, bytes32 s) public view returns (address) {
        return LoomNativeApi.MappedAccount(ethChainId, ecrecover(hash, v, r, s));
    }

    function TestMappedAccount(string memory ethChainId, bytes32 hash, uint8 v, bytes32 r, bytes32 s, string memory toChain) public view returns (address) {
        return LoomNativeApi.MappedAccount(ethChainId, ecrecover(hash, v, r, s), toChain);
    }
}