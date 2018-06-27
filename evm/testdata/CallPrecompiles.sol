pragma solidity ^0.4.21;
contract CallPrecompiles {
    function callPF(uint32 _addr, bytes _input) public view returns (bool) {
        address addr = _addr;
        return addr.call(_input);
    }

    function callPFAssembly(uint64 _addr, bytes _input, uint64 _outLength) public view returns (byte res) {
        address addr = _addr;
        uint256 inSize = _input.length * 4 + 1;
        uint256 outSize = _outLength * 0x20;
        assembly{
            let start := add(_input, 0x04)
            if iszero(call(
                5000,
                addr,
                0,
                start,
                inSize,
                0x40,
                outSize
            )) {
                revert(0,0)
            }
            res := mload(0x40)
            mstore(0x40, add(0x40, outSize))
        }
    }
}