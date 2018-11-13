package hsmpv

import (
	"errors"
	"os"
	"testing"
)

const (
	YHSM_TEST_CONN_URL   = "localhost:12345"
	YHSM_TEST_AUTH_KEYID = 1
	YHSM_TEST_PASSWORD   = "password"
	YHSM_TEST_SIGN_KEYID = 0x0064

	YHSM_TEST_PRIVVAL_CONF = "yhsm_priv_validator.json"
)

// test for init
func TestYubiInit(t *testing.T) {
	if os.Getenv("HSM_YUBICO_TEST_ENABLE") != "true" {
		t.Log("Yubico HSM Test Disabled")
		return
	}

	pv := NewYubiHsmPV(YHSM_TEST_CONN_URL, YHSM_TEST_AUTH_KEYID, YHSM_TEST_PASSWORD, 0)
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

	pv := NewYubiHsmPV(YHSM_TEST_CONN_URL, YHSM_TEST_AUTH_KEYID, YHSM_TEST_PASSWORD, 0)
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

	pv := NewYubiHsmPV(YHSM_TEST_CONN_URL, YHSM_TEST_AUTH_KEYID, YHSM_TEST_PASSWORD, YHSM_TEST_SIGN_KEYID)
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
	if _, err := os.Stat(YHSM_TEST_PRIVVAL_CONF); !os.IsNotExist(err) {
		t.Fatal("HSM priv validator file is already exist. Please try to remove it at first")
	}

	pv := NewYubiHsmPV(YHSM_TEST_CONN_URL, YHSM_TEST_AUTH_KEYID, YHSM_TEST_PASSWORD, 0)
	err := pv.GenPrivVal(YHSM_TEST_PRIVVAL_CONF)
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
	if _, err := os.Stat(YHSM_TEST_PRIVVAL_CONF); os.IsNotExist(err) {
		t.Fatal("No exist HSM priv validator file. Please try genkey at first")
	}

	pv := NewYubiHsmPV(YHSM_TEST_CONN_URL, YHSM_TEST_AUTH_KEYID, YHSM_TEST_PASSWORD, 0)
	err := pv.LoadPrivVal(YHSM_TEST_PRIVVAL_CONF)
	if err != nil {
		t.Fatal(err)
	}
	defer pv.Destroy()

	t.Logf("priv key ID is '%v'", pv.SignKeyID)
}

// sign/verify
func TestYubiSignVerify(t *testing.T) {
	if os.Getenv("HSM_YUBICO_TEST_ENABLE") != "true" {
		t.Log("Yubico HSM Test Disabled")
		return
	}
	var err error

	b := []byte{'t', 'e', 's', 't'}

	pv := NewYubiHsmPV(YHSM_TEST_CONN_URL, YHSM_TEST_AUTH_KEYID, YHSM_TEST_PASSWORD, 0)
	err = pv.LoadPrivVal(YHSM_TEST_PRIVVAL_CONF)
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
