package migrations

import (
	"bytes"

	"github.com/loomnetwork/loomchain/builtin/plugins/chainconfig"
)

func ConsolidateFeaturesMigration(ctx *MigrationContext) error {
	// Pull data from ChainConfig
	_, chainconfigCtx, err := resolveChainConfig(ctx)
	if err != nil {
		return err
	}
	featuresFromState := ctx.state.Range([]byte(featurePrefix))
	for _, m := range featuresFromState {
		data := ctx.state.Get(featureKey(string(m.Key)))
		if bytes.Equal(data, []byte{1}) {
			//Blockheight = 0, means blockHeight is not applicable, as blockheight at which feature was enabled is not known
			chainconfig.SyncEnabledFeature(chainconfigCtx, string(m.Key), 0)
		}
	}
	return nil
}
