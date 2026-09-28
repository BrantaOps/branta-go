package branta

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
)

// Encrypt encrypts value with a key derived from secret.
//
// Wire format MUST match byte-for-byte across every Branta SDK:
//   - Key = SHA-256(UTF-8 secret bytes) — 32 bytes, used directly as the AES-256 key.
//   - Nonce/IV = 12 bytes. Random (CSPRNG) when deterministicNonce is false.
//     When true: the first 12 bytes of HMAC-SHA256(key = key_data [the SHA-256'd
//     key, NOT the raw secret], message = UTF-8 plaintext value).
//   - AES-256-GCM with a 16-byte tag. Wire format: base64_standard(iv || ciphertext || tag).
func Encrypt(value, secret string, deterministicNonce bool) (string, error) {
	keyData := sha256.Sum256([]byte(secret))

	var iv [12]byte
	if deterministicNonce {
		mac := hmac.New(sha256.New, keyData[:])
		mac.Write([]byte(value))
		copy(iv[:], mac.Sum(nil)[:12])
	} else {
		if _, err := io.ReadFull(rand.Reader, iv[:]); err != nil {
			return "", &EncryptionError{Err: err}
		}
	}

	block, err := aes.NewCipher(keyData[:])
	if err != nil {
		return "", &EncryptionError{Err: err}
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", &EncryptionError{Err: err}
	}

	ciphertextAndTag := gcm.Seal(nil, iv[:], []byte(value), nil)

	out := make([]byte, 0, len(iv)+len(ciphertextAndTag))
	out = append(out, iv[:]...)
	out = append(out, ciphertextAndTag...)
	return base64.StdEncoding.EncodeToString(out), nil
}

// Decrypt decrypts a value produced by Encrypt using the matching secret.
//
// A base64 payload shorter than 28 bytes (12 iv + 16 tag minimum) is a distinct
// ErrEncryptedDataTooShort, checked before any crypto is attempted. Any other
// failure (e.g. wrong key -> GCM tag mismatch) collapses into a DecryptionError.
func Decrypt(encryptedValue, secret string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encryptedValue)
	if err != nil {
		return "", &DecryptionError{Err: err}
	}
	if len(data) < 28 {
		return "", ErrEncryptedDataTooShort
	}

	keyData := sha256.Sum256([]byte(secret))
	iv := data[:12]
	ciphertextAndTag := data[12:]

	block, err := aes.NewCipher(keyData[:])
	if err != nil {
		return "", &DecryptionError{Err: err}
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", &DecryptionError{Err: err}
	}
	plaintext, err := gcm.Open(nil, iv, ciphertextAndTag, nil)
	if err != nil {
		return "", &DecryptionError{Err: err}
	}
	return string(plaintext), nil
}

// AESEncryption is the encrypt/decrypt strategy, injectable for testing.
type AESEncryption interface {
	Encrypt(value, secret string, deterministicNonce bool) (string, error)
	Decrypt(encryptedValue, secret string) (string, error)
}

// AESEncryptionService is the production AESEncryption implementation.
type AESEncryptionService struct{}

func (AESEncryptionService) Encrypt(value, secret string, deterministicNonce bool) (string, error) {
	return Encrypt(value, secret, deterministicNonce)
}

func (AESEncryptionService) Decrypt(encryptedValue, secret string) (string, error) {
	return Decrypt(encryptedValue, secret)
}
