package migrations

import (
	"bytes"

	cctypes "github.com/loomnetwork/go-loom/builtin/types/chainconfig"
)

func ConsolidateFeaturesMigration(ctx *MigrationContext) error {
	// Pull data from ChainConfig
	_, chainconfigCtx, err := resolveChainConfig(ctx)
	if err != nil {
		return err
	}
	featuresFromState := ctx.state.Range([]byte(featurePrefix))
	for _, m := range featuresFromState {
		var f cctypes.Feature
		data := ctx.state.Get(featureKey(string(m.Key)))
		if bytes.Equal(data, []byte{1}) {
			f.Status = cctypes.Feature_ENABLED
		} else {
			f.Status = cctypes.Feature_DISABLED
		}
		if err := chainconfigCtx.Set(featureKey(string(m.Key)), &f); err != nil {
			return err
		}
	}
	return nil
}
