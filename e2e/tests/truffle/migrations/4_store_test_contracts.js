const StoreTestContract = artifacts.require('./StoreTestContract.sol')
const TxHashTestContract = artifacts.require('./TxHashTestContract.sol')
const SimpleStoreContract = artifacts.require('./SimpleStore.sol')

module.exports = function (deployer, network, accounts) {
  deployer.then(async () => { 
    await deployer.deploy(StoreTestContract)
    await deployer.deploy(TxHashTestContract)
    await deployer.deploy(SimpleStoreContract)
  })
}
