// +build !gateway
package migrations

func GatewayMigration(ctx *MigrationContext, parameters []byte) error {
	return nil
}
