// Package crypto provides AES-256-GCM encryption for secrets at rest.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

// ErrCiphertextTooShort is returned when the ciphertext is too short to contain IV.
var ErrCiphertextTooShort = errors.New("crypto: ciphertext too short")

// ErrDecryptionFailed is returned when AES-GCM decryption fails.
var ErrDecryptionFailed = errors.New("crypto: decryption failed")

// Encrypt encrypts plaintext using AES-256-GCM with a random IV.
// The IV is prepended to the ciphertext. Returns base64-encoded result.
func Encrypt(plaintext []byte, masterKey string) ([]byte, error) {
	key := deriveKey(masterKey)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return []byte(base64.StdEncoding.EncodeToString(ciphertext)), nil
}

// Decrypt decrypts base64-encoded ciphertext that was encrypted with Encrypt.
// The masterKey must be the same as used for encryption.
func Decrypt(ciphertext []byte, masterKey string) ([]byte, error) {
	key := deriveKey(masterKey)

	// Decode from base64.
	decoded, err := base64.StdEncoding.DecodeString(string(ciphertext))
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(decoded) < nonceSize {
		return nil, ErrCiphertextTooShort
	}

	nonce, ciphertextBytes := decoded[:nonceSize], decoded[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return plaintext, nil
}

// deriveKey derives a 32-byte AES key from the master key using SHA-256.
func deriveKey(masterKey string) []byte {
	hash := sha256.Sum256([]byte(masterKey))
	return hash[:]
}
