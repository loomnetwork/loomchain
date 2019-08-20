const SimpleStore = artifacts.require('./SimpleStore.sol')
const SimpleError = artifacts.require('./SimpleError.sol')

module.exports = function (deployer, network, accounts) {
  deployer.then(async () => { 
    await deployer.deploy(SimpleStore)
    await deployer.deploy(SimpleError)
  })
}
