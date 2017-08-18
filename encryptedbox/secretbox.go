package encryptedbox

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"golang.org/x/crypto/nacl/secretbox"
)

// length of secretbox's key and nonce as defined by secretbox's signature
const keyLength = 32
const nonceLength = 24

// SecretBox presents simple interface for golang.org/x/crypto/nacl/secretbox
type SecretBox interface {

	// SealString will encrypt string, returning hex-encoded version
	// of result
	SealString(message string) (string, error)

	Seal(message []byte) ([]byte, error)

	// OpenString will decrypt hex-encoded string, returning
	// hexencoded version of result
	OpenString(hexmessage string) (string, error)

	Open(message []byte) ([]byte, error)
}

// private implementation of SecretBox
type secretBox struct {
	key [keyLength]byte
}

// NewSecretBox creates SecretBox with key
func NewSecretBox(key string) (SecretBox, error) {
	box := &secretBox{}
	if len(key) != keyLength {
		return nil, fmt.Errorf("incorrect key length (expected %d bytes, got %d)", keyLength, len(key))
	}

	copy(box.key[:], key)

	return box, nil
}

func (b *secretBox) SealString(message string) (string, error) {
	cipher, error := b.Seal([]byte(message))
	if error != nil {
		return "", error
	}
	return hex.EncodeToString(cipher), nil
}

func (b *secretBox) Seal(message []byte) ([]byte, error) {
	n, err := nonce()
	if err != nil {
		return []byte{}, err
	}

	return secretbox.Seal((*n)[:], message, n, &b.key), nil
}

func (b *secretBox) OpenString(hexmessage string) (string, error) {
	message, err := hex.DecodeString(hexmessage)
	if err != nil {
		return "", err
	}

	cipher, err := b.Open(message)
	if err != nil {
		return "", err
	}
	return string(cipher), nil
}

func (b *secretBox) Open(message []byte) ([]byte, error) {
	var nonce [nonceLength]byte
	copy(nonce[:], message[:nonceLength])
	box := message[nonceLength:]

	out := []byte{}

	res, ok := secretbox.Open(out, box, &nonce, &b.key)

	if !ok {
		return []byte{}, errors.New("failed to open secret box")
	}

	return res, nil
}

func nonce() (*[nonceLength]byte, error) {
	var n [nonceLength]byte

	_, err := rand.Read(n[:])

	return &n, err
}
