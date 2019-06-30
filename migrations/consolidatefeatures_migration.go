package migrations

import (
	"github.com/pkg/errors"

	"github.com/gogo/protobuf/proto"
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
		if err := proto.Unmarshal(m.Value, &f); err != nil {
			return errors.Wrapf(err, "unmarshal feature %s", string(m.Key))
		}
		if err := chainconfigCtx.Set(featureKey(f.Name), &f); err != nil {
			return err
		}
	}
	return nil
}

