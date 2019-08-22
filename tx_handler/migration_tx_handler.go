
package tx_handler

import (
	"bytes"
	"encoding/binary"
	"fmt"

	proto "github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"

	loom "github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/util"
	goloomvm "github.com/loomnetwork/go-loom/vm"
	"github.com/loomnetwork/loomchain"
	"github.com/loomnetwork/loomchain/auth"
	"github.com/loomnetwork/loomchain/migrations"
	registry "github.com/loomnetwork/loomchain/registry/factory"
	"github.com/loomnetwork/loomchain/vm"
)

const (
	migrationPrefix = "migrationId"
)

var (
	// ErrFeatureNotEnabled indicates that the migration function feature flag is not enabled
	ErrFeatureNotEnabled = errors.New("[MigrationTxHandler] feature flag is not enabled")
)

func migrationKey(migrationTxID uint32) []byte {
	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, migrationTxID)
	return util.PrefixKey([]byte(migrationPrefix), buf.Bytes())
}

type MigrationFunc func(ctx *migrations.MigrationContext, parameters []byte) error

// MigrationTxHandler handles MigrationTx(s).
type MigrationTxHandler struct {
	Manager        *vm.Manager
	CreateRegistry registry.RegistryFactoryFunc
	Migrations     map[int32]MigrationFunc
}

func (h *MigrationTxHandler) ProcessTx(
	state loomchain.State,
	txBytes []byte,
	isCheckTx bool,
) (loomchain.TxHandlerResult, error) {
	var r loomchain.TxHandlerResult

	if !state.FeatureEnabled(loomchain.MigrationTxFeature, false) {
		return r, fmt.Errorf("MigrationTx feature hasn't been enabled")
	}

	var msg vm.MessageTx
	err := proto.Unmarshal(txBytes, &msg)
	if err != nil {
		return r, err
	}

	origin := auth.Origin(state.Context())
	caller := loom.UnmarshalAddressPB(msg.From)

	if caller.Compare(origin) != 0 {
		return r, fmt.Errorf("Origin doesn't match caller: - %v != %v", origin, caller)
	}

	var tx goloomvm.MigrationTx
	if err := proto.Unmarshal(msg.Data, &tx); err != nil {
		return r, errors.Wrap(err, "failed to unmarshal MigrationTx")
	}

	// allow migration to be run
	migrationRun := state.Get(migrationKey(tx.ID))
	if migrationRun != nil {
		return r, fmt.Errorf("migration ID %d has already been processed", tx.ID)
	}

	id := fmt.Sprint(tx.ID)
	if !state.FeatureEnabled(loomchain.MigrationFeaturePrefix+id, false) {
		return r, fmt.Errorf("feature %s is not enabled", loomchain.MigrationFeaturePrefix+id)
	}

	migrationFn := h.Migrations[int32(tx.ID)]
	if migrationFn == nil {
		return r, fmt.Errorf("invalid migration ID %d", tx.ID)
	}
	migrationCtx := migrations.NewMigrationContext(h.Manager, h.CreateRegistry, state, origin)
	if err := migrationFn(migrationCtx, tx.Input); err != nil {
		return r, errors.Wrapf(err, "migration %d failed", int32(tx.ID))
	}

	state.Set(migrationKey(tx.ID), []byte{1})

	return r, nil
}
