pragma solidity ^0.4.18;
contract TestEvent {
    event MyEvent(uint number);

    function sendEvent(uint i) public   {
        MyEvent(i);
    }

}