package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"
)

func newKey(t *testing.T) string {
	t.Helper()
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("rand: %v", err)
	}
	return base64.StdEncoding.EncodeToString(b)
}

func TestRoundTrip(t *testing.T) {
	c, err := New(newKey(t))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if !c.Enabled() {
		t.Fatal("expected cipher to be enabled")
	}

	plain := "Call me at jane.doe@example.com — ID 123456"
	enc, err := c.Encrypt(plain)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if !strings.HasPrefix(enc, prefix) {
		t.Errorf("ciphertext missing prefix: %q", enc)
	}
	if strings.Contains(enc, "jane.doe") {
		t.Error("plaintext leaked into ciphertext")
	}

	got, err := c.Decrypt(enc)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if got != plain {
		t.Errorf("roundtrip mismatch: got %q want %q", got, plain)
	}
}

func TestNonceIsRandom(t *testing.T) {
	c, _ := New(newKey(t))
	a, _ := c.Encrypt("same")
	b, _ := c.Encrypt("same")
	if a == b {
		t.Error("identical ciphertext for same plaintext — nonce not random")
	}
}

func TestEmptyStaysEmpty(t *testing.T) {
	c, _ := New(newKey(t))
	enc, err := c.Encrypt("")
	if err != nil || enc != "" {
		t.Errorf("empty should stay empty, got %q err %v", enc, err)
	}
}

func TestPassthroughWhenNoKey(t *testing.T) {
	c, err := New("")
	if err != nil {
		t.Fatalf("New(\"\"): %v", err)
	}
	if c.Enabled() {
		t.Fatal("expected passthrough (disabled)")
	}
	enc, _ := c.Encrypt("hello")
	if enc != "hello" {
		t.Errorf("passthrough should not transform, got %q", enc)
	}
	// Plaintext decrypts to itself.
	dec, _ := c.Decrypt("hello")
	if dec != "hello" {
		t.Errorf("passthrough decrypt mismatch, got %q", dec)
	}
}

func TestDecryptPlaintextPrefixless(t *testing.T) {
	c, _ := New(newKey(t))
	// A value stored before encryption was enabled has no prefix.
	got, err := c.Decrypt("legacy plaintext")
	if err != nil || got != "legacy plaintext" {
		t.Errorf("prefixless value should pass through, got %q err %v", got, err)
	}
}

func TestInvalidKeyRejected(t *testing.T) {
	if _, err := New("not-base64!!!"); err == nil {
		t.Error("expected error for non-base64 key")
	}
	short := base64.StdEncoding.EncodeToString(make([]byte, 16))
	if _, err := New(short); err == nil {
		t.Error("expected error for 16-byte key (need 32)")
	}
}

func TestWrongKeyFails(t *testing.T) {
	c1, _ := New(newKey(t))
	c2, _ := New(newKey(t))
	enc, _ := c1.Encrypt("secret")
	if _, err := c2.Decrypt(enc); err == nil {
		t.Error("expected decrypt to fail with wrong key")
	}
}

func TestTamperDetected(t *testing.T) {
	c, _ := New(newKey(t))
	enc, _ := c.Encrypt("secret")
	// Flip a character in the base64 body.
	body := []byte(enc)
	last := len(body) - 1
	if body[last] == 'A' {
		body[last] = 'B'
	} else {
		body[last] = 'A'
	}
	if _, err := c.Decrypt(string(body)); err == nil {
		t.Error("expected tampered ciphertext to fail authentication")
	}
}
