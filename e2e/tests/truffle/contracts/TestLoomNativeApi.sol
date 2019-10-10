pragma solidity >=0.4.21;

import "./LoomNativeApi.sol";

library TestLoomNativeApi {
    function TestMappedLoomAccount(string memory fromChain, string memory hash, uint8 v, bytes32 r, bytes32 s) public view returns (address) {
        return LoomNativeApi.MappedAccount(fromChain, ecrecover(stripPrefix(hash), v, r, s));
    }

    function TestMappedAccount(string memory fromChain, string memory hash, uint8 v, bytes32 r, bytes32 s, string memory toChain) public view returns (address) {
        return LoomNativeApi.MappedAccount(fromChain, ecrecover(stripPrefix(hash), v, r, s), toChain);
    }

    function stripPrefix(string memory message) private view returns (bytes32) {
        string memory prefix = "\x19Ethereum Signed Message:\n000000";
        uint256 lengthOffset;
        uint256 length;
        assembly {
            length := mload(message)
            lengthOffset := add(prefix, 57)
        }
        require(length <= 999999);
        uint256 lengthLength = 0;
        uint256 divisor = 100000;
        while (divisor != 0) {
            uint256 digit = length / divisor;
            if (digit == 0) {
                // Skip leading zeros
                if (lengthLength == 0) {
                    divisor /= 10;
                    continue;
                }
            }
            lengthLength++;
            length -= digit * divisor;
            divisor /= 10;
            digit += 0x30;
            lengthOffset++;
            assembly {
                mstore8(lengthOffset, digit)
            }
        }
        if (lengthLength == 0) {
            lengthLength = 1 + 0x19 + 1;
        } else {
            lengthLength += 1 + 0x19;
        }
        assembly {
            mstore(prefix, lengthLength)
        }
        return keccak256(abi.encodePacked(prefix, message));
    }
}
