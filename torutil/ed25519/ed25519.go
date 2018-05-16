// Package ed25519 implements Tor/BitTorrent-like ed25519 keys.
//
// See the following stack overflow post for details on why
// golang.org/x/crypto/ed25519 can't be used:
//  https://stackoverflow.com/questions/44810708/ed25519-public-result-is-different
package ed25519

import (
	"crypto"
	"crypto/rand"
	"crypto/sha512"
	"errors"
	"io"

	"github.com/cretz/bine/torutil/ed25519/internal/edwards25519"
	"golang.org/x/crypto/ed25519"
)

// Ref: https://stackoverflow.com/questions/44810708/ed25519-public-result-is-different

// PrivateKey is a 64-byte Ed25519 private key.
type PrivateKey []byte

// PublicKey is a 32-byte Ed25519 public key.
type PublicKey []byte

// FromCryptoPrivateKey converts a Go private key to the one in this package.
func FromCryptoPrivateKey(key ed25519.PrivateKey) PrivateKey {
	digest := sha512.Sum512(key[:32])
	digest[0] &= 248
	digest[31] &= 127
	digest[31] |= 64
	return digest[:]
}

// FromCryptoPublicKey converts a Go public key to the one in this package.
func FromCryptoPublicKey(key ed25519.PublicKey) PublicKey {
	return PublicKey(key)
}

func (p PrivateKey) Public() crypto.PublicKey {
	return p.PublicKey()
}

func (p PrivateKey) PublicKey() PublicKey {
	var A edwards25519.ExtendedGroupElement
	var hBytes [32]byte
	copy(hBytes[:], p[:])
	edwards25519.GeScalarMultBase(&A, &hBytes)
	var publicKeyBytes [32]byte
	A.ToBytes(&publicKeyBytes)
	return publicKeyBytes[:]
}

func (p PrivateKey) Sign(rand io.Reader, message []byte, opts crypto.SignerOpts) (signature []byte, err error) {
	if opts.HashFunc() != crypto.Hash(0) {
		return nil, errors.New("ed25519: cannot sign hashed message")
	}
	panic("TODO")
}

// GenerateKey generates a public/private key pair using entropy from rand.
// If rand is nil, crypto/rand.Reader will be used.
func GenerateKey(rnd io.Reader) (publicKey PublicKey, privateKey PrivateKey, err error) {
	if rnd == nil {
		rnd = rand.Reader
	}
	_, err = io.ReadFull(rnd, privateKey[:32])
	if err == nil {
		digest := sha512.Sum512(privateKey[:32])
		digest[0] &= 248
		digest[31] &= 127
		digest[31] |= 64
		privateKey = digest[:]
		publicKey = privateKey.PublicKey()
	}
	return
}
