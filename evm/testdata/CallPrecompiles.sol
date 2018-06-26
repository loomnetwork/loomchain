pragma solidity ^0.4.21;
contract CallPrecompiles {
    bytes public pcfResult;

    function callPF(uint32 _addr, bytes _input) public view returns (bool) {
        address addr = _addr;
        return addr.call(_input);
    }

    function callPFAssembly(uint32 _addr, bytes input) public {
        address addr = _addr;
        //uint256[4] memory input;
        bool rtv;
        bytes memory callResult;
        assembly{
            rtv := call(0, addr, 0, input, 0xFFFF, callResult, 0xFFFF)
        }
        pcfResult = callResult;
    }
}