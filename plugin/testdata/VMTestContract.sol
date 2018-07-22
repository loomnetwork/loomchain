pragma solidity ^0.4.24;

contract VMTestContract {
    address public expectedCaller;
    address public lastCaller;

    function setExpectedCaller(address _caller) public {
        expectedCaller = _caller;
    }

    function checkTxCaller() public
    {
        require(msg.sender == expectedCaller);
        lastCaller = msg.sender;
    }

    function checkQueryCaller() public view returns (address)
    {
        return msg.sender;
    }
}
