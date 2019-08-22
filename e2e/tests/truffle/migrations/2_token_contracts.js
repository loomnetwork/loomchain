const MyToken = artifacts.require('./MyToken.sol')
const MyCoin = artifacts.require('./MyCoin.sol')
const TestEvent = artifacts.require('./TestEvent.sol')
const ChainTestEvent = artifacts.require('./ChainTestEvent.sol')

module.exports = function (deployer, network, accounts) {
  deployer.then(async () => {
    await deployer.deploy(MyToken)
    await deployer.deploy(MyCoin)
    await deployer.deploy(TestEvent)
    await deployer.deploy(ChainTestEvent)
  })
}
