package karma

import (
	"encoding/hex"
	"testing"
	"github.com/loomnetwork/go-loom"
	"github.com/loomnetwork/go-loom/plugin/contractpb"
	"github.com/loomnetwork/go-loom/plugin"
	"github.com/stretchr/testify/require"
	"github.com/loomnetwork/loomchain/builtin/plugins/karma/types"
)

var (
	valPubKeyHex1 = "3866f776276246e4f9998aa90632931d89b0d3a5930e804e02299533f55b39e1"
)

func TestKarmaGetState(t *testing.T) {
	smsKarma := int64(10)
	oathKarma := int64(10)
	tokeKarma := int64(1)
	c := &Karma{}
	pubKey, _ := hex.DecodeString(valPubKeyHex1)
	addr := loom.Address{
		Local: loom.LocalAddressFromPublicKey(pubKey),
	}
	ctx := contractpb.WrapPluginContext(
		plugin.CreateFakeContext(addr, addr),
	)

	config := &Config{
		SmsKarma:           smsKarma,
		OauthKarma:        	oathKarma,
		TokenKarma: 		tokeKarma,
		LastUpdateTime: 	ctx.Now().Unix(),
	}

	ctx.Set(configKey, config)

	params := &karma.KarmaOwner{
		Owner: "Owner",
	}
	resp, err := c.GetConfig(ctx, params)

	require.Nil(t, err)
	require.Equal(t, smsKarma, resp.SmsKarma)
	require.Equal(t, oathKarma, resp.OauthKarma)
	require.Equal(t, tokeKarma, resp.TokenKarma)

}
