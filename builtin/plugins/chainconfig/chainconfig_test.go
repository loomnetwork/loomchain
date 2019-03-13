package chainconfig

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ChainConfigTestSuite struct {
}

func (c *ChainConfigTestSuite) SetupTest() {
}

func TestChainConfigTestSuite(t *testing.T) {
	suite.Run(t, new(ChainConfigTestSuite))
}

func (c *ChainConfigTestSuite) TestFeatureFlagEnabled() {
}
