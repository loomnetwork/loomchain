const fs = require('fs')
const path = require('path')
const LoomTruffleProvider = require('loom-truffle-provider')
const HDWalletProvider = require("@truffle/hdwallet-provider")

module.exports = {
  networks: {
    local: {
      provider: function() {
        if (!process.env.CLUSTER_DIR) {
          throw new Error('CLUSTER_DIR env var not defined')
        }
        const nodeAddr = fs.readFileSync(path.join(process.env.CLUSTER_DIR, '0', 'node_rpc_addr'), 'utf-8').trim()
        console.log(`Using node at ${nodeAddr} for Truffle`)
        const chainId = 'default'
        const writeUrl = `http://${nodeAddr}/rpc`
        const readUrl = `http://${nodeAddr}/query`
        const privateKey = fs.readFileSync(path.join(__dirname, 'private_key'), 'utf-8')
        const provider = new LoomTruffleProvider(chainId, writeUrl, readUrl, privateKey)
        provider.createExtraAccountsFromMnemonic("gravity top burden flip student usage spell purchase hundred improve check genre", 10)
        return provider
      },
      network_id: '*',
      skipDryRun: true,
    },
    hdwallet: {
      provider: function() {
        if (!process.env.CLUSTER_DIR) {
          throw new Error('CLUSTER_DIR env var not defined')
        }
        const nodeAddr = fs.readFileSync(path.join(process.env.CLUSTER_DIR, '0', 'node_rpc_addr'), 'utf-8').trim()
        console.log(`Using node at ${nodeAddr} for Truffle`)
        const mnemonic = fs.readFileSync(path.join(__dirname, 'eth_mnemonic'), 'utf-8').trim()
        // NOTE: This provider uses Eth accounts, so a mapping to a Loom account must already for
        // any account used with this provider.
        return new HDWalletProvider(mnemonic, `http://${nodeAddr}/eth`, 0, 6)
      },
      network_id: '*',
      skipDryRun: true,
    }
  }
}
