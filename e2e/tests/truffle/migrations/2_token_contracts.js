const MyToken = artifacts.require('./MyToken.sol')
const MyCoin = artifacts.require('./MyCoin.sol')
const InnerEmitter = artifacts.require('./InnerEmitter.sol')
const OuterEmitter = artifacts.require('./OuterEmitter.sol')

module.exports = function (deployer, network, accounts) {
  deployer.then(async () => {
    await deployer.deploy(MyToken)
    await deployer.deploy(MyCoin)
    await deployer.deploy(InnerEmitter)
    await deployer.deploy(OuterEmitter)
  })
}
