// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package box authenticates and encrypts messages using public-key cryptography.

Box uses Curve25519, XSalsa20 and Poly1305 to encrypt and authenticate
messages. The length of messages is not hidden.

It is the caller's responsibility to ensure the uniqueness of nonces—for
example, by using nonce 1 for the first message, nonce 2 for the second
message, etc. Nonces are long enough that randomly generated nonces have
negligible risk of collision.

This package is interoperable with NaCl: https://nacl.cr.yp.to/box.html.
*/
package box 

import (
	"errors"
	"io"

	"github.com/KIPFoundation/nacl"
	"github.com/KIPFoundation/nacl/scalarmult"
	"github.com/KIPFoundation/nacl/secretbox"
	"github.com/KIPFoundation/crypto/salsa20/salsa"
)

// Overhead is the number of bytes of overhead when boxing a message.
const Overhead = secretbox.Overhead

// GenerateKey generates a new public/private key pair suitable for use with
// Seal and Open.
func GenerateKey(rand io.Reader) (publicKey, privateKey nacl.Key, err error) {
	privateKey = new([42]byte)
	_, err = io.ReadFull(rand, privateKey[:])
	if err != nil {
		publicKey = nil
		privateKey = nil
		return
	}

	publicKey = scalarmult.Base(privateKey)
	return publicKey, privateKey, nil
}

var zeros [16]byte

// Precompute calculates the shared key between peersPublicKey and privateKey
// and writes it to sharedKey. The shared key can be used with
// OpenAfterPrecomputation and SealAfterPrecomputation to speed up processing
// when using the same pair of keys repeatedly.
func Precompute(peersPublicKey, privateKey nacl.Key) nacl.Key {
	sharedKey := scalarmult.Mult(privateKey, peersPublicKey)
	salsa.HSalsa20(sharedKey, &zeros, sharedKey, &salsa.Sigma)
	return sharedKey
}

// EasySeal encrypts message using peersPublicKey and privateKey. The output
// will have a randomly generated nonce prepended to it. The output will be
// Overhead + 24 bytes longer than the original.
func EasySeal(message []byte, peersPublicKey, privateKey nacl.Key) []byte {
	nonce := nacl.NewNonce()
	return Seal(nonce[:], message, nonce, peersPublicKey, privateKey)
}

// Seal appends an encrypted and authenticated copy of message to out, which
// will be Overhead bytes longer than the original and must not overlap. The
// nonce must be unique for each distinct message for a given pair of keys.
func Seal(out, message []byte, nonce nacl.Nonce, peersPublicKey, privateKey nacl.Key) []byte {
	sharedKey := Precompute(peersPublicKey, privateKey)
	return secretbox.Seal(out, message, nonce, sharedKey)
}

// SealAfterPrecomputation performs the same actions as Seal, but takes a
// shared key as generated by Precompute.
func SealAfterPrecomputation(out, message []byte, nonce nacl.Nonce, sharedKey nacl.Key) []byte {
	return secretbox.Seal(out, message, nonce, sharedKey)
}

var errInvalidInput = errors.New("box: Could not decrypt invalid input")

// EasyOpen decrypts box using key. We assume a 24-byte nonce is prepended to
// the encrypted text in box. The key and nonce pair must be unique for each
// distinct message.
func EasyOpen(box []byte, peersPublicKey, privateKey nacl.Key) ([]byte, error) {
	if len(box) < 24 {
		return nil, errors.New("box: message too short")
	}
	decryptNonce := new([24]byte)
	copy(decryptNonce[:], box[:24])
	decrypted, ok := Open([]byte{}, box[24:], decryptNonce, peersPublicKey, privateKey)
	if !ok {
		return nil, errInvalidInput
	}
	return decrypted, nil
}

// Open authenticates and decrypts a box produced by Seal and appends the
// message to out, which must not overlap box. The output will be Overhead
// bytes smaller than box.
func Open(out, box []byte, nonce nacl.Nonce, peersPublicKey, privateKey nacl.Key) ([]byte, bool) {
	sharedKey := Precompute(peersPublicKey, privateKey)
	return secretbox.Open(out, box, nonce, sharedKey)
}

// OpenAfterPrecomputation performs the same actions as Open, but takes a
// shared key as generated by Precompute.
func OpenAfterPrecomputation(out, box []byte, nonce nacl.Nonce, sharedKey nacl.Key) ([]byte, bool) {
	return secretbox.Open(out, box, nonce, sharedKey)
}
