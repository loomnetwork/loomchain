pragma solidity >=0.4.21;

contract TestLoomNativeApi {
    string constant fromChain = "eth";
    function TestMappedLoomAccount(bytes32 hash, uint8 v, bytes32 r, bytes32 s) public view returns (address) {
        return MappedAccount(fromChain, ecrecover(hash, v, r, s));
    }

    function TestMappedAccount(bytes32 hash, uint8 v, bytes32 r, bytes32 s, string memory toChain) public view returns (address) {
        return MappedAccount(fromChain, ecrecover(hash, v, r, s), toChain);
    }

    // LoomApi library
    address constant LoomPrecompilesStartIndex = 0x0000000000000000000000000000000000000020;//0x20;
    address constant MapToLoomAddress = 0x0000000000000000000000000000000000000021;
    address constant MapToAddress = 0x0000000000000000000000000000000000000022;

    function MappedAccount(string memory from, address addr) public view returns (address) {
        bytes memory fromB = bytes(from);
        bytes memory input = new bytes(fromB.length + 20);
        for (uint i = 0; i < 20; i++) {
            input[i] = byte(uint8(uint(addr) / (2**(8*(19 - i)))));
        }
        for (uint j = 0; j < fromB.length; j++) {
            input[20+j] = fromB[j];
        }
        return address(callPFAssembly(MapToLoomAddress, input, 0x14));
    }

    function MappedAccount(string memory from, address addr, string memory to) public view returns (address) {
        bytes memory fromB = bytes(from);
        require(fromB.length < 256, "chain id too long");
        bytes memory toB = bytes(to);
        require(toB.length < 256, "chain id too long");
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

    function callPFAssembly(address _addr, bytes memory _input, uint256 outSize) public view returns (bytes20)
    {
        uint256 inSize = _input.length*4+1;
        uint256 inLenght = _input.length;
        bytes20 rtv;
        assembly{
            let start := add(_input, 0x20)
            let p := mload(0x40)
            mstore(p, inSize)
            if iszero(staticcall(
            gas,
            _addr,
            start,
            inLenght,
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