package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateName(t *testing.T) {
	assert.Nil(t, validateName("foo123"))
	assert.Nil(t, validateName("foo.bar"))

	assert.NotNil(t, validateName("foo@bar"))
}
