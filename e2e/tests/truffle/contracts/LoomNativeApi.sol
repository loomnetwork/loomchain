pragma solidity >=0.4.21;

library LoomNativeApi {
    address constant LoomPrecompilesStartIndex = 0x0000000000000000000000000000000000000020;
    address constant MapToLoomAddress = 0x0000000000000000000000000000000000000021;
    address constant MapToAddress = 0x0000000000000000000000000000000000000022;

    // Calls MapToLoomAddress precompiled EVM function.this.
    // Returns the loom address mapped to the input local addr and fromChainId..
    function MappedAccount(string memory fromChainId, address addr) public view returns (address) {
        bytes memory fromB = bytes(fromChainId);
        bytes memory input = new bytes(fromB.length + 20);
        // Encode from chain id and local address into bytes object for passing to precompied function
        // [<addr - 20 bytes>, <from chain id, rest of array>]"
        for (uint i = 0; i < 20; i++) {
            input[i] = byte(uint8(uint(addr) / (2**(8*(19 - i)))));
        }
        for (uint j = 0; j < fromB.length; j++) {
            input[20+j] = fromB[j];
        }
        return address(callPFAssembly(MapToLoomAddress, input, 0x14));
    }

    // Calls MapToAddress precompiled EVM function.this.
    // Returns the address with toChainId mapped to the input local addr and fromChainId.
    function MappedAccount(string memory fromChainId, address addr, string memory toChainId) public view returns (address) {
        // restrict from chain id to length 256 so as to hold length in one byte.
        bytes memory fromB = bytes(fromChainId);
        require(fromB.length < 256, "chain id too long");
        bytes memory toB = bytes(toChainId);
        require(toB.length < 256, "chain id too long");

        // Encode from and to chain ids and local address into bytes object for passing to precompied function
        // [<addr - 20 bytes>, <length of from chain id, 1 byte>, <from chain id>, <to chain id, rest of array>]"
        bytes memory input = new bytes(fromB.length + toB.length + 21);
        for (uint i = 0; i < 20; i++) {
            input[i] = byte(uint8(uint(addr) / (2**(8*(19 - i)))));
        }
        input[20] = byte(uint8(fromB.length));
        for (uint j = 0; j < fromB.length; j++) {
            input[21+j] = fromB[j];
        }
        for (uint k = 0; k < toB.length; k++) {
            input[21+fromB.length+k] = toB[k];
        }
        return address(callPFAssembly(MapToAddress, input, 0x14));
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
