const NonceTestContract = artifacts.require('./NonceTestContract.sol')

module.exports = function (deployer, network, accounts) {
  deployer.then(async () => { 
    await deployer.deploy(NonceTestContract)
  })
}
