pragma solidity ^0.4.21;
contract CallPrecompiles {
    function callPF(uint32 _addr, bytes _input) public view returns (bool) {
        address addr = _addr;
        return addr.call(_input);
    }

    function callPFAssembly(uint64 _addr, bytes _input, uint64 _outLength) public view returns (bytes rtv) {
        address addr = _addr;
        uint256 inSize = _input.length * 8;
        uint256 outSize = _outLength * 0x20;
        //bytes memory callResult = new bytes(_outLength);
        uint256[2] memory callResult;
        assembly{
            if iszero(call(
            5000,
            addr,
            0,
            _input,
            inSize,
            callResult,
            0x40
            )) {
                revert(0,0)
            }
        }
        return callResult;
    }
}