// +build !gateway

package migrations

import "github.com/pkg/errors"

// GatewayMigration is a placeholder used in non-gateway builds.
func GatewayMigration(ctx *MigrationContext, parameters []byte) error {
	return errors.New("This is not implemented")
}
