const EventTestContract = artifacts.require('./EventTestContract.sol')

module.exports = function (deployer, network, accounts) {
  deployer.then(async () => { 
    await deployer.deploy(EventTestContract)
  })
}
