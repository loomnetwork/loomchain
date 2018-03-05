package ledger

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExchange(t *testing.T) {
	assert, require := assert.New(t), require.New(t)

	cases := []struct {
		input string
	}{
		{"food"},
		{"q4w35ertdyfugihojpkdryftughiuj"},
		{
			`this is a very long message... oh so long
      that it splits over many many lines.
      q43w5e65rtiyuoporaestdyfugihoijrdytfuygih
      weurityuoisyrdutfiuyoio5w4e6r7t8y9udrytfuygiuhij`,
		},
	}

	for i, tc := range cases {
		data := []byte(tc.input)

		// no 0x9000 trailer...
		echo := NewLedger(NewEcho(64))
		resp, err := echo.Exchange(data, 100)
		require.NotNil(err)

		// note: we need to append 9000 for success
		echo = NewLedger(NewEcho(64))
		send := append(data, 0x90, 0x0)
		resp, err = echo.Exchange(send, 100)
		require.Nil(err, "%d: %+v", i, err)
		assert.Equal(data, resp, "%d", i)
	}
}
