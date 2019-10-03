pragma solidity >=0.4.21;

contract TestLoomApi {
    function TestMappedAccount(bytes32 hash, uint8 v, bytes32 r, bytes32 s) public view returns (address) {
        return MappedAccount("eth", ecrecover(hash, v, r, s));
    }

    // LoomApi library
    address constant LoomPrecompilesStartIndex = 0x0000000000000000000000000000000000000020;//0x20;
    address constant MapToLoomAddress = 0x0000000000000000000000000000000000000021;
    address constant MapAddresses = 0x0000000000000000000000000000000000000022;

    function MappedAccount(string memory from, address addr) public view returns (address) {
        bytes memory fromB = bytes(from);
        bytes memory input = new bytes(fromB.length + 20);
        for (uint i = 0; i < 20; i++) {
            input[i] = byte(uint8(uint(addr) / (2**(8*(19 - i)))));
        }
        //https://ethereum.stackexchange.com/questions/884/how-to-convert-an-address-to-bytes-in-solidity/885#885
        for (uint j = 0; j < fromB.length; j++) {
            input[20+j] = fromB[j];
        }
        return address(callPFAssembly(MapToLoomAddress, input));
    }

    //function MappedAccount(string from, address addr, string to) public view returns (address) {
    //    return this;
    //}

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

}