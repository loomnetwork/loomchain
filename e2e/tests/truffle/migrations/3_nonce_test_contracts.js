const NonceTestContract = artifacts.require('./NonceTestContract.sol')
const OutOfMemoryContract = artifacts.require('./OutOfMemory.sol')

module.exports = function (deployer, network, accounts) {
  deployer.then(async () => { 
    await deployer.deploy(NonceTestContract)
    await deployer.deploy(OutOfMemoryContract)
  })
}
