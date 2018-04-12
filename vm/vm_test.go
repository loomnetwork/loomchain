package vm

import (
	"testing"
)

func TestProcessDeployTx(t *testing.T) {
	testEvents(t)
	testLoomTokens(t)
	testCryptoZombies(t)
}

