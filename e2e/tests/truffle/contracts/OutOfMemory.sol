pragma solidity ^0.5.0;

contract OutOfMemory {
    uint256 proofRequest;
    uint256 zkpPrime;
mapping (address => string) public accounts;

constructor() public{
    proofRequest = uint256(sha256('save'));
    zkpPrime = uint256(11);
}
    event Show(address _caller,bytes32 _commandHash,uint256 _proofReq);
    event Show2(bytes32 _command,uint256 _proofRequest,uint256 _commandHash);
    function renderHelloWorld () public pure returns (string memory) {
        return 'helloWorld';
 }

  function storeTransactionP1(bytes32 command)public{
   // emit Show2(command,proofRequest,bytes32(proofRequest));
     bytes32 commandHash = sha256(abi.encodePacked(command));
        emit Show2(command,proofRequest,uint256(commandHash));
    // emit Show(msg.sender,commandHash,proofRequest);
    // require(proofRequest % (uint256(commandHash) & (2**128 - 1)) == 0, 'Wrong hash check');
   //  return proofRequest % (uint256(commandHash) & (2**128 - 1)) == 0;
  }

  function getProofRequest()public view returns(uint256){
    return proofRequest;
  }
}
