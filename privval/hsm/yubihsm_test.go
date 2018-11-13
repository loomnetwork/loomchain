package hsmpv

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"testing"
)

const (
	DefaultYubiConnURL   = "127.0.0.1:12345"
	DefaultYubiAuthKeyID = 1
	DefaultYubiPassword  = "password"

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
}

// parse YubiHSM info
func parseYubiHSMInfo(t *testing.T) (*YubiHsmInfo, error) {
	// check if testing for Yubico has been enabled
	if os.Getenv("HSM_YUBICO_TEST_ENABLE") != "true" {
		t.Fatal("Yubico HSM Test Disabled")
	}

	// create new instance of YubiHsmInfo with default value
	yubiHsmInfo := &YubiHsmInfo{
		ConnURL:    DefaultYubiConnURL,
		AuthKeyID:  DefaultYubiAuthKeyID,
		AuthPasswd: DefaultYubiPassword,
	}

	// check if priv validator is exist
	if _, err := os.Stat(YubiHsmInfoConf); os.IsNotExist(err) {
		return yubiHsmInfo, nil
	}

	t.Log("Reading YubiHSM configuration info")

	// parse priv validator file
	jsonBytes, err := ioutil.ReadFile(YubiHsmInfoConf)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(jsonBytes, yubiHsmInfo)
	if err != nil {
		return nil, err
	}

	return yubiHsmInfo, nil
}

// test for init
func TestYubiInit(t *testing.T) {
	yubiHsmInfo, err := parseYubiHSMInfo(t)
	if err != nil {
		t.Fatal(err)
	}

	pv := NewYubiHsmPV(yubiHsmInfo.ConnURL, yubiHsmInfo.AuthKeyID, yubiHsmInfo.AuthPasswd, 0)
	err = pv.Init()
	if err != nil {
		t.Fatal(err)
	}
	pv.Destroy()
}

// test for genkey
func TestYubiGenkey(t *testing.T) {
	yubiHsmInfo, err := parseYubiHSMInfo(t)
	if err != nil {
		t.Fatal(err)
	}

	pv := NewYubiHsmPV(yubiHsmInfo.ConnURL, yubiHsmInfo.AuthKeyID, yubiHsmInfo.AuthPasswd, 0)
	err = pv.Init()
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
	yubiHsmInfo, err := parseYubiHSMInfo(t)
	if err != nil {
		t.Fatal(err)
	}

	if yubiHsmInfo.SignKeyID == 0x00 {
		t.Fatal(errors.New("Please specify sign key ID in config"))
	}

	pv := NewYubiHsmPV(yubiHsmInfo.ConnURL, yubiHsmInfo.AuthKeyID, yubiHsmInfo.AuthPasswd, yubiHsmInfo.SignKeyID)
	err = pv.Init()
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
	yubiHsmInfo, err := parseYubiHSMInfo(t)
	if err != nil {
		t.Fatal(err)
	}

	pv := NewYubiHsmPV(yubiHsmInfo.ConnURL, yubiHsmInfo.AuthKeyID, yubiHsmInfo.AuthPasswd, 0)
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
	yubiHsmInfo, err := parseYubiHSMInfo(t)
	if err != nil {
		t.Fatal(err)
	}

	pv := NewYubiHsmPV(yubiHsmInfo.ConnURL, yubiHsmInfo.AuthKeyID, yubiHsmInfo.AuthPasswd, 0)
	err = pv.LoadPrivVal(YubiHsmPrivValConf)
	if err != nil {
		t.Fatal(err)
	}
	defer pv.Destroy()

	t.Logf("priv key ID is '%v'", pv.SignKeyID)
}

// sign/verify
func TestYubiSignVerify(t *testing.T) {
	b := []byte{'t', 'e', 's', 't'}

	yubiHsmInfo, err := parseYubiHSMInfo(t)
	if err != nil {
		t.Fatal(err)
	}

	pv := NewYubiHsmPV(yubiHsmInfo.ConnURL, yubiHsmInfo.AuthKeyID, yubiHsmInfo.AuthPasswd, 0)
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
