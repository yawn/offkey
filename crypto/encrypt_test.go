package crypto_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yawn/offkey/crypto"
)

func TestEncrypt(t *testing.T) {

	assert := assert.New(t)

	var (
		passphrase = "open sesame"
		secret     []byte
	)

	for _, size := range []int{64, 128, 256, 512, 1024, 1489} {

		secret = make([]byte, size)

		_, err := crypto.Encrypt(passphrase, secret)
		assert.NoError(err)

	}

	secret = make([]byte, 1490)

	_, err := crypto.Encrypt(passphrase, secret)
	assert.EqualError(err, "secret too large - 1490 exceed maximum size of 1489")

}
