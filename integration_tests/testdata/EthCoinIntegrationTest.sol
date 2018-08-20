pragma solidity ^0.4.24;

contract EthCoinIntegrationTest {
    // transfer ETH from the caller to the contract
    function deposit() external payable {
        address(this).transfer(msg.value);
    }

    // transfer ETH from the contract to the caller
    function withdraw(uint256 amount) external {
        msg.sender.transfer(amount);
    }

    function withdrawThenFail(uint256 amount) external {
        msg.sender.transfer(amount);
        revert();
    }

    function failThenWithdraw(uint256 amount) external {
        revert();
        msg.sender.transfer(amount);
    }

    // transfer ETH from the caller to the recipient
    function transfer(address recipient) external payable {
        recipient.transfer(msg.value);
    }

    function transferThenFail(address recipient) external payable {
        recipient.transfer(msg.value);
        revert();
    }

    function failThenTransfer(address recipient) external payable {
        revert();
        recipient.transfer(msg.value);
    }

    // get the current ETH balance of the specified account
    function getBalance(address account) external view returns (uint256) {
        return account.balance;
    }
}
