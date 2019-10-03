pragma solidity >=0.4.21;
contract CallPrecompiles {
    function callPFAssembly(address _addr, bytes memory _input) public view returns (bytes20)
    {
        uint256 inSize = _input.length*4+1;
        uint256 inLenght = _input.length;
        uint256 outSize = 0x20;
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

    /*function callPFAssembly(address _addr, bytes memory _input) public view returns (bytes32)
    {
        uint256 inSize = _input.length*4+1;
        uint256 aaaaa = input.length*3;
        uint256 outSize = 0x20;
        bytes32 rtv;
        assembly{
            let start := add(_input, 0x04)
            if iszero(staticcall(
            gas,
            _addr,
            start,
            inSize,
            rtv,
            outSize
            )) {
                revert(0,0)
            }
            mstore(0x40, add(0x40, outSize))
        }
        return rtv;
    }*/
}