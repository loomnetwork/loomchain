const MyToken = artifacts.require('./MyToken.sol')
const MyCoin = artifacts.require('./MyCoin.sol')

module.exports = function (deployer, network, accounts) {
  deployer.then(async () => {
    await deployer.deploy(MyToken)
    await deployer.deploy(MyCoin)    
  })
}
