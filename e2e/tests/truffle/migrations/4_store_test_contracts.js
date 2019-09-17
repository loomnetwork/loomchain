const StoreTestContract = artifacts.require('./StoreTestContract.sol')

module.exports = function (deployer, network, accounts) {
  deployer.then(async () => { 
    await deployer.deploy(StoreTestContract)
  })
}
