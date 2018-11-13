package hsmpv

import (
	"errors"
	"os"
	"testing"
)

const (
	YubiHsmConnURL   = "localhost:12345"
	YubiHsmAuthKeyID = 1
	YubiHsmPassword  = "password"
	YubiHsmSignKeyID = 0x0064

	YubiHsmPrivValConf = "yhsm_priv_validator.json"
)

// test for init
func TestYubiInit(t *testing.T) {
	if os.Getenv("HSM_YUBICO_TEST_ENABLE") != "true" {
		t.Log("Yubico HSM Test Disabled")
		return
	}

	pv := NewYubiHsmPV(YubiHsmConnURL, YubiHsmAuthKeyID, YubiHsmPassword, 0)
	err := pv.Init()
	if err != nil {
		t.Fatal(err)
	}
	pv.Destroy()
}

// test for genkey
func TestYubiGenkey(t *testing.T) {
	if os.Getenv("HSM_YUBICO_TEST_ENABLE") != "true" {
		t.Log("Yubico HSM Test Disabled")
		return
	}

	pv := NewYubiHsmPV(YubiHsmConnURL, YubiHsmAuthKeyID, YubiHsmPassword, 0)
	err := pv.Init()
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
	if os.Getenv("HSM_YUBICO_TEST_ENABLE") != "true" {
		t.Log("Yubico HSM Test Disabled")
		return
	}

	pv := NewYubiHsmPV(YubiHsmConnURL, YubiHsmAuthKeyID, YubiHsmPassword, YubiHsmSignKeyID)
	err := pv.Init()
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
	if os.Getenv("HSM_YUBICO_TEST_ENABLE") != "true" {
		t.Log("Yubico HSM Test Disabled")
		return
	}

	// check if priv validator is exist
	if _, err := os.Stat(YubiHsmPrivValConf); !os.IsNotExist(err) {
		t.Fatal("HSM priv validator file is already exist. Please try to remove it at first")
	}

	pv := NewYubiHsmPV(YubiHsmConnURL, YubiHsmAuthKeyID, YubiHsmPassword, 0)
	err := pv.GenPrivVal(YubiHsmPrivValConf)
	if err != nil {
		t.Fatal(err)
	}
	pv.Save()
	pv.Destroy()

	t.Logf("priv key ID is '%v'", pv.SignKeyID)
}

// load YubiHSM priv validator
func TestYubiLoadHsm(t *testing.T) {
	if os.Getenv("HSM_YUBICO_TEST_ENABLE") != "true" {
		t.Log("Yubico HSM Test Disabled")
		return
	}
	// check if priv validator is exist
	if _, err := os.Stat(YubiHsmPrivValConf); os.IsNotExist(err) {
		t.Fatal("No exist HSM priv validator file. Please try genkey at first")
	}

	pv := NewYubiHsmPV(YubiHsmConnURL, YubiHsmAuthKeyID, YubiHsmPassword, 0)
	err := pv.LoadPrivVal(YubiHsmPrivValConf)
	if err != nil {
		t.Fatal(err)
	}
	defer pv.Destroy()

	t.Logf("priv key ID is '%v'", pv.SignKeyID)
}

// sign/verify
func TestYubiSignVerify(t *testing.T) {
	var err error
	if os.Getenv("HSM_YUBICO_TEST_ENABLE") != "true" {
		t.Log("Yubico HSM Test Disabled")
		return
	}

	b := []byte{'t', 'e', 's', 't'}

	pv := NewYubiHsmPV(YubiHsmConnURL, YubiHsmAuthKeyID, YubiHsmPassword, 0)
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
