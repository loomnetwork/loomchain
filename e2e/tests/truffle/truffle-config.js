const fs = require('fs')
const path = require('path')
const LoomTruffleProvider = require('loom-truffle-provider')

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
    }
  }
}
