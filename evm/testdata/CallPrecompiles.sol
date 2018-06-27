pragma solidity ^0.4.21;
contract CallPrecompiles {
    uint256[2] public pcfResult;

    function callPF(uint32 _addr, bytes _input) public view returns (bool) {
        address addr = _addr;
        return addr.call(_input);
    }

    function callPFAssembly(uint32 _addr) public view returns (uint256[2] rtv) {
        address addr = _addr;
        uint256[2] memory callResult;
        uint256[4] memory _input;
        assembly{
            if iszero(call(not(0), addr, 0, _input, 0x80, callResult, 0x40)) {
                revert(0,0)
            }
        }
        return callResult;
    }
}