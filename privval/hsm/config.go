package hsmpv

const (
	HSM_DEV_TYPE_SOFT  = "softhsm"
	HSM_DEV_TYPE_YUBI  = "yubihsm"
	HSM_DEV_TYPE_CLOUD = "cloudhsm"
)

// HSM device configuration
type HsmConfig struct {
	HsmEnabled bool

	// device type of HSM
	HsmDevType string

	// the path of PKCS#11 library
	HsmP11LibPath string

	// device login credential
	HsmDevLoginCred string

	// connection URL to YubiHSM
	HsmConnUrl string

	// Auth key ID for YubiHSM
	HsmAuthKeyId uint16
}

func DefaultConfig() *HsmConfig {
	return &HsmConfig{
		HsmEnabled:      false,
		HsmDevType:      "softhsm",
		HsmP11LibPath:   "/usr/local/lib/softhsm/libsofthsm2.so",
		HsmDevLoginCred: "123456",
	}
}
