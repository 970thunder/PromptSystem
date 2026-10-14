package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// encryptedValuePrefix marks an at-rest encrypted system_settings value.
// Values without the prefix are treated as legacy plaintext and pass through
// unchanged, so enabling encryption never breaks existing rows; they are
// re-encrypted the next time the setting is saved.
const encryptedValuePrefix = "enc:v1:"

// SettingsCipher provides AES-256-GCM at-rest encryption for sensitive
// system_settings values (currently only smtp.password). The key is provision
// specific and lives outside the database (injected via the environment from
// the deployment secret file), so a database dump alone does not expose the
// credentials inside.
type SettingsCipher struct {
	aead cipher.AEAD
}

// ErrInvalidSettingsCipher reports a malformed or missing key material.
var ErrInvalidSettingsCipher = errors.New("invalid settings cipher key")

// NewSettingsCipher accepts a 32-byte raw key or its base64 encoding.
// Shorter passphrases are stretched with SHA-256 for operator convenience.
func NewSettingsCipher(key string) (*SettingsCipher, error) {
	if key == "" {
		return nil, ErrInvalidSettingsCipher
	}
	material, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		material = []byte(key)
	}
	if len(material) == 0 {
		return nil, ErrInvalidSettingsCipher
	}
	if len(material) != 32 {
		digest := sha256.Sum256(material)
		material = digest[:]
	}
	block, err := aes.NewCipher(material)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSettingsCipher, err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidSettingsCipher, err)
	}
	return &SettingsCipher{aead: aead}, nil
}

// Encrypt seals the plaintext under the provision key. Empty values are
// returned unchanged so "clear this setting" semantics stay in the store.
func (c *SettingsCipher) Encrypt(plaintext string) (string, error) {
	if c == nil || plaintext == "" {
		return plaintext, nil
	}
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return encryptedValuePrefix + base64.StdEncoding.EncodeToString(sealed), nil
}

// Decrypt opens a prefixed value. Legacy plaintext passes through unchanged.
// A value that carries the prefix but fails to open decrypts to an empty
// string: the affected send fails with an obvious SMTP auth error and the
// administrator re-saves the credential, instead of the whole settings read
// erroring out.
func (c *SettingsCipher) Decrypt(stored string) string {
	if c == nil || !IsEncryptedValue(stored) {
		return stored
	}
	raw, err := base64.StdEncoding.DecodeString(stored[len(encryptedValuePrefix):])
	if err != nil || len(raw) < c.aead.NonceSize() {
		return ""
	}
	nonce, sealed := raw[:c.aead.NonceSize()], raw[c.aead.NonceSize():]
	plaintext, err := c.aead.Open(nil, nonce, sealed, nil)
	if err != nil {
		return ""
	}
	return string(plaintext)
}

// IsEncryptedValue reports whether the stored value carries the cipher prefix.
func IsEncryptedValue(stored string) bool {
	return len(stored) > len(encryptedValuePrefix) && stored[:len(encryptedValuePrefix)] == encryptedValuePrefix
}
