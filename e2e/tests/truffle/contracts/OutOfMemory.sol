pragma solidity ^0.5.0;

contract OutOfMemory {
    uint256 proofRequest;
    uint256 zkpPrime;
    uint256 mulModValue;
    string[] dynamicSizeArray;
    address owner;

constructor() public{
    zkpPrime = uint256(899863899863899863899863899863899863899863899863899863);
    owner = msg.sender;
    proofRequest = uint256(sha256(abi.encodePacked("save")));
}
    event ShowHashAfterAnd(bytes32 andOperHash);
    event Show(address _caller,bytes32 _commandHash,uint256 _proofReq);
    event ShowHash(bytes32 command,bytes32 commandHashByte,uint256 commandHashUint);
    event HashCheck(bool hashCheck);
    event MulModCheck(bool mulMod);
    event DebugUint(bytes32 value);

    modifier onlyOwner {
    require(msg.sender == owner);
    _;
}

function getProofRequest()public view returns(uint256){
    return proofRequest;
  }

    function renderHelloWorld () public pure returns (string memory) {
        return 'helloWorld';
 }

 function brokenArray()onlyOwner public {
   for  (uint32 i = 0 ; i < 2**32-1 ; i++) {
     dynamicSizeArray.push("require(proofRequest % (uint256(commandHash) & (2**128 - 1)) == 0, 'Wrong hash check');");
    }
 }

  function HashAndMod(bytes32 command) onlyOwner public {
    bytes32 commandHash = sha256(abi.encodePacked(command));
    uint256 andOperHash = uint256(commandHash) & 2**128-1;
    emit ShowHashAfterAnd(bytes32(andOperHash));
    emit HashCheck(uint256(commandHash) % andOperHash == 0);
  }

  function MultiplyModulo(bytes32 command) public{
    bytes32 commandHash = sha256(abi.encodePacked(command));
    uint256 n = uint256(commandHash) * zkpPrime;
    uint256 m = uint256(command) * zkpPrime;
    emit DebugUint(bytes32(n));
    emit DebugUint(bytes32(m));
    emit DebugUint(bytes32(zkpPrime));
    emit DebugUint(bytes32(mulmod(n, m, zkpPrime)));
  }
}
