package hsmpv

import (
	"os"
	"os/exec"

	"testing"
)

const (
	SHSM_P11LIB_PATH       = "/usr/local/lib/softhsm/libsofthsm2.so"
	SHSM_TEST_CONF         = "hsm_test.json"

	SHSM_TEST_SLOT_NUM     = "0"
	SHSM_TEST_TOKEN_LABLE  = "loomchain"
	SHSM_TEST_TOKEN_SOPIN  = "123456"
	SHSM_TEST_TOKEN_PIN    = "123456"

	SHSM_TEST_PRIVVAL_CONF = "shsm_priv_validator.json"
)

// test for init
func TestSoftInit(t *testing.T) {
	var cmd *exec.Cmd

	// delete token
	t.Logf("Deleting old soft token object")
	cmd = exec.Command("softhsm2-util", "--delete-token", "--token", SHSM_TEST_TOKEN_LABLE)

	// remove priv validator file
	t.Logf("Removing old HSM priv validator configuration")
	os.Remove(SHSM_TEST_PRIVVAL_CONF)
	cmd.Run()

	// init token
	t.Logf("Initializing soft token")
	cmd = exec.Command("softhsm2-util", "--init-token", "--slot", SHSM_TEST_SLOT_NUM, "--label", SHSM_TEST_TOKEN_LABLE,
		"--so-pin", SHSM_TEST_TOKEN_SOPIN, "--pin", SHSM_TEST_TOKEN_PIN)
	err := cmd.Run()
	if err != nil {
		t.Fatal(err)
	}
}

// test for genkey
func TestSoftGenkey(t *testing.T) {
	// check if priv validator is exist
	if _, err := os.Stat(SHSM_TEST_PRIVVAL_CONF); !os.IsNotExist(err) {
		t.Fatal("HSM priv validator file is already exist. Please try to init token at first")
	}

	pv := NewSoftHsmPV(SHSM_P11LIB_PATH, SHSM_TEST_TOKEN_PIN)
	err := pv.GenPrivVal(SHSM_TEST_PRIVVAL_CONF)
	if err != nil {
		t.Fatal(err)
	}
	pv.Save()
	pv.Destroy()

	t.Logf("priv key ID is '%v'", pv.SignKeyID)
	t.Logf("public key is '%v'", pv.PubKeyBytes.Bytes())
}

// load HSM priv validator
func TestSoftLoadHsm(t *testing.T) {
	// check if priv validator is exist
	if _, err := os.Stat(SHSM_TEST_PRIVVAL_CONF); os.IsNotExist(err) {
		t.Fatal("No exist HSM priv validator file. Please try genkey at first")
	}

	pv := NewSoftHsmPV(SHSM_P11LIB_PATH, SHSM_TEST_TOKEN_PIN)
	err := pv.LoadPrivVal(SHSM_TEST_PRIVVAL_CONF)
	if err != nil {
		t.Fatal(err)
	}
	defer pv.Destroy()

	t.Logf("priv key ID is '%v'", pv.SignKeyID)
	t.Logf("public key is '%v'", pv.PubKeyBytes.Bytes())
}

// sign/verify
func TestSoftSignVerify(t *testing.T) {
	var sig []byte
	var err error

	b := []byte{'t', 'e', 's', 't'}

	pv := NewSoftHsmPV(SHSM_P11LIB_PATH, SHSM_TEST_TOKEN_PIN)
	err = pv.LoadPrivVal(SHSM_TEST_PRIVVAL_CONF)
	if err != nil {
		t.Fatal(err)
	}
	defer pv.Destroy()

	sig, err = pv.signBytes(b)
	if err != nil {
		t.Fatal(err)
	}

	err = pv.verifySign(b, sig)
	if err != nil {
		t.Fatal(err)
	}
}