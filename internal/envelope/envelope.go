// Package envelope encrypts user data with AES-256-GCM using two levels of keys.
// A master key wraps each user's data key, and the data key encrypts the user's
// content. Associated data binds every ciphertext to its owner and record, so
// ciphertexts can't be moved between users or rows without detection.
package envelope

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
)

const KeySize = 32

var ErrDecrypt = errors.New("envelope: decryption failed")

type Keyring struct {
	master []byte
}

func NewKeyring(master []byte) (*Keyring, error) {
	if len(master) != KeySize {
		return nil, fmt.Errorf("envelope: master key must be %d bytes", KeySize)
	}
	return &Keyring{master: master}, nil
}

// NewDataKey returns a random key for encrypting one user's content.
func NewDataKey() ([]byte, error) {
	return randomBytes(KeySize)
}

func (k *Keyring) WrapDataKey(dek, aad []byte) ([]byte, error) {
	return seal(k.master, dek, aad)
}

func (k *Keyring) UnwrapDataKey(wrapped, aad []byte) ([]byte, error) {
	return open(k.master, wrapped, aad)
}

func Encrypt(dek, plaintext, aad []byte) ([]byte, error) {
	return seal(dek, plaintext, aad)
}

func Decrypt(dek, ciphertext, aad []byte) ([]byte, error) {
	return open(dek, ciphertext, aad)
}

// seal returns nonce || ciphertext.
func seal(key, plaintext, aad []byte) ([]byte, error) {
	aead, err := newAEAD(key)
	if err != nil {
		return nil, err
	}
	nonce, err := randomBytes(aead.NonceSize())
	if err != nil {
		return nil, err
	}
	return aead.Seal(nonce, nonce, plaintext, aad), nil
}

func open(key, sealed, aad []byte) ([]byte, error) {
	aead, err := newAEAD(key)
	if err != nil {
		return nil, err
	}
	n := aead.NonceSize()
	if len(sealed) < n {
		return nil, ErrDecrypt
	}
	plaintext, err := aead.Open(nil, sealed[:n], sealed[n:], aad)
	if err != nil {
		return nil, ErrDecrypt
	}
	return plaintext, nil
}

func newAEAD(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return nil, err
	}
	return b, nil
}
