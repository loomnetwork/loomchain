pragma solidity ^0.4.21;
contract GlobalProperties {
    function Blockhash(uint blockNumber) view public returns (bytes32) {
        return block.blockhash(blockNumber);
    }

    function blockCoinbase() view public returns (address) {
        return block.coinbase;
    }

    function blockDifficulty() view public returns (uint) {
        return block.difficulty;
    }

    function blockGasLimit() view public returns (uint) {
        return block.gaslimit;
    }

    function blockNumber() view public returns (uint) {
        return block.number;
    }

    function blockTimeStamp() view public returns (uint) {
        return block.timestamp;
    }

    function msgData() pure public returns (bytes) {
        return msg.data;
    }

    function gasLeft() view public returns (uint) {
        return gasleft();
    }

    function msgSender() view public returns (address) {
        return msg.sender;
    }

    function msgSig() pure  public returns (bytes4) {
        return msg.sig;
    }

    function msgValue() view  public returns (uint) {
        return msg.value;
    }

    function Now() view public returns (uint) {
        return now;
    }

    function txGasPrice() view public returns (uint) {
        return tx.gasprice;
    }

    function txOrigin() view public returns (address) {
        return tx.origin;
    }
}