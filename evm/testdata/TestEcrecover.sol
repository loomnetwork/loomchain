pragma solidity ^0.4.21;

contract TestEcrecover {
    function QueryEcrecover(bytes32 hash, uint8 v, bytes32 r, bytes32 s) public pure returns (address signer){
        return ecrecover(hash, v, r, s);
    }

    function QueryEcrecoverPrefix(bytes32 hash, uint8 v, bytes32 r, bytes32 s) public pure returns (address signer){
        bytes memory prefix = "\x19Ethereum Signed Message:\n32";
        bytes32 prefixedHash = keccak256(prefix, hash);
        return ecrecover(prefixedHash, v, r, s);
    }
}

