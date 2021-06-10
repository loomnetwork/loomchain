const LoomNativeApi = artifacts.require("./LoomNativeApi.sol");
const TestLoomNativeApi = artifacts.require("./TestLoomNativeApi.sol");

module.exports = function(deployer) {
    deployer.deploy(LoomNativeApi);
    deployer.link(LoomNativeApi, TestLoomNativeApi);
    deployer.deploy(TestLoomNativeApi);
};
