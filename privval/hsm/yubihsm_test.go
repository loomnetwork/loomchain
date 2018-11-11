package hsmpv

import (
	"os"
	"errors"
	"testing"
)

const (
	YHSM_TEST_CONN_URL    = "localhost:1234"
	YHSM_TEST_AUTH_KEYID  = 0
	YHSM_TEST_PASSWORD    = "123456"

	YHSM_TEST_PRIVVAL_CONF = "yhsm_priv_validator.json"
)

// test for genkey
func TestYubiGenkey(t *testing.T) {
	// check if priv validator is exist
	if _, err := os.Stat(YHSM_TEST_PRIVVAL_CONF); !os.IsNotExist(err) {
		t.Fatal("HSM priv validator file is already exist. Please try to remove it at first")
	}

	pv := NewYubiHsmPV(YHSM_TEST_CONN_URL, YHSM_TEST_AUTH_KEYID, YHSM_TEST_PASSWORD)
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
	// check if priv validator is exist
	if _, err := os.Stat(SHSM_TEST_PRIVVAL_CONF); os.IsNotExist(err) {
		t.Fatal("No exist HSM priv validator file. Please try genkey at first")
	}

	pv := NewYubiHsmPV(YHSM_TEST_CONN_URL, YHSM_TEST_AUTH_KEYID, YHSM_TEST_PASSWORD)
	err := pv.LoadPrivVal(YHSM_TEST_PRIVVAL_CONF)
	if err != nil {
		t.Fatal(err)
	}
	defer pv.Destroy()

	t.Logf("priv key ID is '%v'", pv.SignKeyID)
}

// sign/verify
func TestYubiSignVerify(t *testing.T) {
	var err error

	b := []byte{'t', 'e', 's', 't'}

	pv := NewYubiHsmPV(YHSM_TEST_CONN_URL, YHSM_TEST_AUTH_KEYID, YHSM_TEST_PASSWORD)
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
		t.Fatal(errors.New("Verifying signation has failed."))
	}
}
