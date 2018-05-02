pragma solidity ^0.4.18;
contract TestEvent {
    event MyEvent(uint number, uint value);
    event TestEvent2(address indexed from, uint value);
    event Transfer(string indexed from, address indexed to, uint value);
    function sendEvent(uint i, uint value) public   {
        MyEvent(i, value);
    }
    function sendEvents(uint number, uint value, string from) public {
        MyEvent(number, value);
        TestEvent2(msg.sender, value);
        Transfer(from, msg.sender, value);
    }

}