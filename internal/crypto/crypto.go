// Package crypto provides authenticated encryption for short PII field values
// (feedback comment, timezone) stored in SQLite. It uses AES-256-GCM.
//
// Threat model: this protects data at rest — a leaked backup, a stolen database
// file, or a volume snapshot. The key is supplied at runtime (FEEDBACK_ENC_KEY),
// so an attacker with full access to the running host can still read both the key
// and the data; that is out of scope, the same as for whole-file schemes like
// SQLCipher.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

// prefix marks an encrypted, encoded value. Stored values without it are treated
// as plaintext (written while encryption was disabled), so the two can coexist
// and a passthrough reader still works.
const prefix = "enc:v1:"

// Cipher encrypts and decrypts field values. A Cipher built from an empty key is
// a passthrough (Enabled()==false): it returns values unchanged, for local dev
// and tests where no key is configured.
type Cipher struct {
	aead cipher.AEAD // nil => passthrough
}

// New builds a Cipher from a base64-encoded 32-byte key (AES-256). An empty key
// returns a passthrough Cipher. A non-empty but malformed key is an error, so
// callers fail closed rather than silently storing plaintext under a bad key.
func New(keyB64 string) (*Cipher, error) {
	keyB64 = strings.TrimSpace(keyB64)
	if keyB64 == "" {
		return &Cipher{}, nil
	}

	key, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		return nil, fmt.Errorf("FEEDBACK_ENC_KEY is not valid base64: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("FEEDBACK_ENC_KEY must decode to 32 bytes (AES-256), got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Cipher{aead: aead}, nil
}

// Enabled reports whether encryption is active (a key was configured).
func (c *Cipher) Enabled() bool { return c.aead != nil }

// Encrypt returns an encoded ciphertext for plaintext. Empty input returns empty
// (so empty columns stay empty). A passthrough Cipher returns plaintext as-is.
func (c *Cipher) Encrypt(plaintext string) (string, error) {
	if plaintext == "" || c.aead == nil {
		return plaintext, nil
	}

	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}
	// Seal prepends nonce (the dst arg) to the ciphertext+tag.
	sealed := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return prefix + base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt reverses Encrypt. A value without the encryption prefix (a plaintext
// row written while encryption was disabled) is returned unchanged.
func (c *Cipher) Decrypt(stored string) (string, error) {
	if !strings.HasPrefix(stored, prefix) {
		return stored, nil
	}
	if c.aead == nil {
		return "", errors.New("value is encrypted but no FEEDBACK_ENC_KEY is set")
	}

	raw, err := base64.StdEncoding.DecodeString(stored[len(prefix):])
	if err != nil {
		return "", fmt.Errorf("invalid ciphertext encoding: %w", err)
	}
	ns := c.aead.NonceSize()
	if len(raw) < ns {
		return "", errors.New("ciphertext too short")
	}
	nonce, ct := raw[:ns], raw[ns:]

	pt, err := c.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt failed (wrong key or tampered data): %w", err)
	}
	return string(pt), nil
}
