package crypto

import (
	"bytes"
	"fmt"

	"filippo.io/age"
	"filippo.io/age/armor"
	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
	"github.com/pkg/errors"
)

const (
	eTooLarge = "secret too large - %d exceed maximum size of 1489"
)

// Encrypt encrypts a secret into a barcode, using age armor as payload
func Encrypt(pass string, secret []byte) (barcode.Barcode, error) {

	if l := len(secret); l > 1489 {
		return nil, fmt.Errorf(eTooLarge, l)
	}

	rec, err := age.NewScryptRecipient(pass)

	if err != nil {
		return nil, errors.Wrapf(err, "failed to create scrypt recipient")
	}

	w1 := bytes.NewBuffer(nil)
	w2 := armor.NewWriter(w1)

	w3, err := age.Encrypt(w2, rec)

	if err != nil {
		return nil, errors.Wrapf(err, "failed to encrypt secret")
	}

	if _, err := w3.Write(secret); err != nil {
		return nil, errors.Wrapf(err, "failed to armor secret")
	}

	if err := w3.Close(); err != nil {
		return nil, errors.Wrapf(err, "failed to finalize encryption")
	}

	if err := w2.Close(); err != nil {
		return nil, errors.Wrapf(err, "failed to finalize armor")
	}

	code, err := qr.Encode(w1.String(), qr.M, qr.Auto)

	if err != nil {
		return nil, errors.Wrapf(err, "failed to encode armor to qr")
	}

	return code, nil

}
