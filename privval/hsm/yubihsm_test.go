package hsmpv

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"testing"

	lcrypto "github.com/loomnetwork/go-loom/crypto"
)

const (
	YubiHsmInfoConf    = "yhsm_test.json"
	YubiHsmPrivValConf = "yhsm_priv_validator.json"
)

type YubiHsmInfo struct {
	// connection URL
	ConnURL string `json:"ConnURL"`
	// auth Key ID
	AuthKeyID uint16 `json:"AuthKeyID"`
	// auth password
	AuthPasswd string `json:"AuthPasswd"`
	// sign keyID
	SignKeyID uint16 `json:"SignKeyID"`
	// sign key domain
	SignKeyDomain uint16 `json:"SignKeyDomain"`
}

// parse YubiHSM info
func parseYubiHSMInfo(t *testing.T) (*HsmConfig, error) {
	hsmConfig := DefaultConfig()

	// check if priv validator is exist
	if _, err := os.Stat(YubiHsmInfoConf); os.IsNotExist(err) {
		return hsmConfig, nil
	}

	t.Log("Reading YubiHSM configuration info")

	// parse priv validator file
	jsonBytes, err := ioutil.ReadFile(YubiHsmInfoConf)
	if err != nil {
		return nil, err
	}

	yubiHsmInfo := &YubiHsmInfo{}
	err = json.Unmarshal(jsonBytes, yubiHsmInfo)
	if err != nil {
		return nil, err
	}

	if len(yubiHsmInfo.ConnURL) > 0 {
		hsmConfig.HsmConnURL = yubiHsmInfo.ConnURL
	}

	if yubiHsmInfo.AuthKeyID > 0 {
		hsmConfig.HsmAuthKeyID = yubiHsmInfo.AuthKeyID
	}

	if len(yubiHsmInfo.AuthPasswd) > 0 {
		hsmConfig.HsmAuthPassword = yubiHsmInfo.AuthPasswd
	}

	if yubiHsmInfo.SignKeyID > 0 {
		hsmConfig.HsmSignKeyID = yubiHsmInfo.SignKeyID
	}

	if yubiHsmInfo.SignKeyDomain > 0 {
		hsmConfig.HsmSignKeyDomain = yubiHsmInfo.SignKeyDomain
	}

	return hsmConfig, nil
}

// test for init
func TestYubiInit(t *testing.T) {
	// check if testing for Yubico has been enabled
	if os.Getenv("HSM_YUBICO_TEST_ENABLE") != "true" {
		t.Log("Yubico HSM Test Disabled")
		return
	}

	hsmConfig, err := parseYubiHSMInfo(t)
	if err != nil {
		t.Fatal(err)
	}

	pv := NewYubiHsmPrivVal(hsmConfig)
	pv.PrivateKey, err = lcrypto.InitYubiHsmPrivKey(pv.hsmConfig)
	if err != nil {
		t.Fatal(err)
	}
	pv.Destroy()
}

// test for genkey
func TestYubiGenkey(t *testing.T) {
	// check if testing for Yubico has been enabled
	if os.Getenv("HSM_YUBICO_TEST_ENABLE") != "true" {
		t.Log("Yubico HSM Test Disabled")
		return
	}

	hsmConfig, err := parseYubiHSMInfo(t)
	if err != nil {
		t.Fatal(err)
	}

	pv := NewYubiHsmPrivVal(hsmConfig)
	pv.PrivateKey, err = lcrypto.InitYubiHsmPrivKey(pv.hsmConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer pv.Destroy()

	err = pv.genEd25519KeyPair()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("sign key ID is '%v'", pv.SignKeyID)
}

// test for exportkey
func TestYubiExportkey(t *testing.T) {
	// check if testing for Yubico has been enabled
	if os.Getenv("HSM_YUBICO_TEST_ENABLE") != "true" {
		t.Log("Yubico HSM Test Disabled")
		return
	}

	hsmConfig, err := parseYubiHSMInfo(t)
	if err != nil {
		t.Fatal(err)
	}

	pv := NewYubiHsmPrivVal(hsmConfig)
	pv.PrivateKey, err = lcrypto.InitYubiHsmPrivKey(pv.hsmConfig)
	if err != nil {
		t.Fatal(err)
	}
	defer pv.Destroy()

	err = pv.exportEd25519PubKey()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("sign key ID is '%v'", pv.SignKeyID)
}

// test for gen priv validator
func TestYubiGenPrivval(t *testing.T) {
	// check if testing for Yubico has been enabled
	if os.Getenv("HSM_YUBICO_TEST_ENABLE") != "true" {
		t.Log("Yubico HSM Test Disabled")
		return
	}

	hsmConfig, err := parseYubiHSMInfo(t)
	if err != nil {
		t.Fatal(err)
	}

	pv := NewYubiHsmPrivVal(hsmConfig)
	err = pv.GenPrivVal(YubiHsmPrivValConf)
	if err != nil {
		t.Fatal(err)
	}
	pv.Save()
	pv.Destroy()

	t.Logf("priv key ID is '%v'", pv.SignKeyID)
}

// load YubiHSM priv validator
func TestYubiLoadHsm(t *testing.T) {
	// check if testing for Yubico has been enabled
	if os.Getenv("HSM_YUBICO_TEST_ENABLE") != "true" {
		t.Log("Yubico HSM Test Disabled")
		return
	}

	hsmConfig, err := parseYubiHSMInfo(t)
	if err != nil {
		t.Fatal(err)
	}

	pv := NewYubiHsmPrivVal(hsmConfig)
	err = pv.LoadPrivVal(YubiHsmPrivValConf)
	if err != nil {
		t.Fatal(err)
	}
	defer pv.Destroy()

	t.Logf("priv key ID is '%v'", pv.SignKeyID)
}

// sign/verify
func TestYubiSignVerify(t *testing.T) {
	// check if testing for Yubico has been enabled
	if os.Getenv("HSM_YUBICO_TEST_ENABLE") != "true" {
		t.Log("Yubico HSM Test Disabled")
		return
	}

	b := []byte{'t', 'e', 's', 't'}

	hsmConfig, err := parseYubiHSMInfo(t)
	if err != nil {
		t.Fatal(err)
	}

	pv := NewYubiHsmPrivVal(hsmConfig)
	err = pv.LoadPrivVal(YubiHsmPrivValConf)
	if err != nil {
		t.Fatal(err)
	}
	defer pv.Destroy()

	sig, err := pv.signBytes(b)
	if err != nil {
		t.Fatal(err)
	}

	if pv.verifySig(b, sig) != true {
		t.Fatal(errors.New("verifying signation has failed"))
	}
}
