// +build !gateway

package migrations

import "github.com/pkg/errors"

func GatewayMigration(ctx *MigrationContext, parameters []byte) error {
	return errors.New("This is not implemented")
}
