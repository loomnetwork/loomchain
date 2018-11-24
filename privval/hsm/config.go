package hsmpv

// HSM device types
const (
	HsmDevTypeSoft  = "softhsm"
	HsmDevTypeYubi  = "yubihsm"
	HsmDevTypeCloud = "cloudhsm"
)

// HsmConfig implements configurations for HSM device
type HsmConfig struct {
	// flag to enable HSM
	HsmEnabled bool

	// device type of HSM
	HsmDevType string

	// the path of PKCS#11 library
	HsmP11LibPath string

	// device login credential
	HsmDevLoginCred string

	// connection URL to YubiHSM
	HsmConnURL string

	// Auth key ID for YubiHSM
	HsmAuthKeyID uint16

	// Sign Key ID for YubiHSM
	HsmSignKeyID uint16
}

// DefaultConfig creates new instance of HsmConfig with default config
func DefaultConfig() *HsmConfig {
	return &HsmConfig{
		HsmEnabled:      false,
		HsmDevType:      "yubihsm",
		HsmP11LibPath:   "",
		HsmDevLoginCred: "password",
		HsmConnURL:      "http://localhost:12345",
		HsmAuthKeyID:    1,
	}
}
