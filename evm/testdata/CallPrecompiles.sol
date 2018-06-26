pragma solidity ^0.4.21;
contract CallPrecompiles {
    bytes public pcfResult;

    function callPF(uint32 _addr, bytes _input) public view returns (bool) {
        address addr = _addr;
        return addr.call(_input);
    }

    function callPFAssembly(uint32 _addr, bytes input) public {
        address addr = _addr;
        bool success;
        bytes memory callResult;
        byte[4] memory _input;
        _input[0] = input[0];
        assembly{
            success := call(0, addr, 0, _input, 0x80, callResult, 0x40)
        }
        pcfResult = callResult;
    }
}