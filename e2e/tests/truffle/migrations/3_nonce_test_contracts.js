const NonceTest = artifacts.require('./NonceTest.sol')

module.exports = function (deployer, network, accounts) {
  deployer.then(async () => { 
    await deployer.deploy(NonceTest)
  })
}
