pragma solidity >=0.4.21;

contract TestLoomNativeApi {
    function TestMappedLoomAccount(string memory ethChainId, bytes32 hash, uint8 v, bytes32 r, bytes32 s) public view returns (address) {
        return MappedAccount(ethChainId, ecrecover(hash, v, r, s));
    }

    function TestMappedAccount(string memory ethChainId, bytes32 hash, uint8 v, bytes32 r, bytes32 s, string memory toChain) public view returns (address) {
        return MappedAccount(ethChainId, ecrecover(hash, v, r, s), toChain);
    }

    // LoomApi library
    address constant MapToAddress = 0x0000000000000000000000000000000000000021;
    uint constant addressLength = 0x14;

    // Calls MapToLoomAddress precompiled EVM function.this.
    // Returns the loom address mapped to the input local addr and fromChainId..
    function MappedAccount(string memory fromChainId, address addr) public view returns (address) {
        // restrict from chain id to length 256 so as to hold length in one byte.
        bytes memory fromB = bytes(fromChainId);
        require(fromB.length < 256, "chain id too long");
        bytes memory empty;
        return address(callPFAssembly(MapToAddress, packInput(addr, fromB, empty), addressLength));
    }

    // Calls MapToAddress precompiled EVM function.this.
    // Returns the address with toChainId mapped to the input local addr and fromChainId.
    function MappedAccount(string memory fromChainId, address addr, string memory toChainId) public view returns (address) {
        // restrict from chain id to length 256 so as to hold length in one byte.
        bytes memory fromB = bytes(fromChainId);
        require(fromB.length < 256, "chain id too long");
        bytes memory toB = bytes(toChainId);
        require(toB.length < 256, "chain id too long");

        return address(callPFAssembly(MapToAddress, packInput(addr, fromB,  toB), addressLength));
    }

    // Encode from and to chain ids and local address into bytes object for passing to pre-complied function
    // [<addr - 20 bytes>, <length of from chain id, 1 byte>, <from chain id>, <optional to chain id, rest of array>]
    function packInput(address addr, bytes memory fromChainId,  bytes memory toChainId) pure internal returns (bytes memory) {
        bytes memory input = new bytes(fromChainId.length + toChainId.length + 21);

        //convert address to bytes
        for (uint i = 0; i < 20; i++) {
            input[i] = byte(uint8(uint(addr) / (2**(8*(19 - i)))));
        }

        input[20] = byte(uint8(fromChainId.length));
        for (uint j = 0; j < fromChainId.length; j++) {
            input[21+j] = fromChainId[j];
        }
        for (uint k = 0; k < toChainId.length; k++) {
            input[21+fromChainId.length+k] = toChainId[k];
        }

        return input;
    }

    // Call precompiled EVM function at address _addr.
    // Pass though _input as input parameter.
    function callPFAssembly(address _addr, bytes memory _input, uint256 outSize) view internal returns (bytes20)    {
        uint256 inSize = _input.length*4+1;
        uint256 inLength = _input.length;
        bytes20 rtv;
        assembly{
            let start := add(_input, 0x20)
            let p := mload(0x40)
            mstore(p, inSize)
            if iszero(staticcall(
            gas,
            _addr,
            start,
            inLength,
            p,
            outSize
            )) {
                revert(0,0)
            }
            rtv := mload(p)
        }
        return rtv;
    }
}