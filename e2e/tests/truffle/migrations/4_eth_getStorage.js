const SimpleStorage = artifacts.require('./SimpleStorage.sol')

module.exports = function (deployer, network, accounts) {
  deployer.then(async () => { 
    await deployer.deploy(SimpleStorage)
  })
}
