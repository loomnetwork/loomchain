pragma solidity ^0.4.21;
contract CallPrecompiles {
    function callPF(uint32 _addr, bytes _input) public view returns (bool) {
        address addr = _addr;
        return addr.call(_input);
    }

    function callPFAssembly(uint64 _addr, bytes _input, uint64 _outLength) public view returns (uint256[20] p) {
        address addr = _addr;
        uint256 inSize = _input.length * 4 + 1;
        uint256 outSize = _outLength * 0x20;
        bytes memory res = new bytes(_outLength);
        assembly{
            let start := add(_input, 0x04)
            let resData := add(res, 0x20)
            if iszero(call(
            5000,
            addr,
            0,
            start,
            inSize,
            p,
            outSize
            )) {
                revert(0,0)
            }
            mstore(0x40, add(0x40, outSize))
        }
    }
}