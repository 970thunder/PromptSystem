package store

import (
	"strings"
	"testing"
)

func TestSettingsCipherRoundTrip(t *testing.T) {
	cipher, err := NewSettingsCipher("unit-test-passphrase")
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}

	sealed, err := cipher.Encrypt("smtp-secret")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if !IsEncryptedValue(sealed) {
		t.Fatalf("ciphertext missing prefix: %q", sealed)
	}
	if strings.Contains(sealed, "smtp-secret") {
		t.Fatal("ciphertext leaks plaintext")
	}
	if got := cipher.Decrypt(sealed); got != "smtp-secret" {
		t.Fatalf("decrypt = %q", got)
	}

	// Empty values pass through so store-level "clear" semantics stay intact.
	if sealed, err := cipher.Encrypt(""); err != nil || sealed != "" {
		t.Fatalf("empty encrypt = %q err=%v", sealed, err)
	}

	// Legacy plaintext rows pass through untouched.
	if got := cipher.Decrypt("legacy-plaintext"); got != "legacy-plaintext" {
		t.Fatalf("legacy passthrough = %q", got)
	}

	// A foreign key cannot open the value: result is empty, not an error.
	other, err := NewSettingsCipher("another-passphrase")
	if err != nil {
		t.Fatal(err)
	}
	if got := other.Decrypt(sealed); got != "" {
		t.Fatalf("foreign key decrypt = %q, want empty", got)
	}

	if _, err := NewSettingsCipher(""); err != ErrInvalidSettingsCipher {
		t.Fatalf("empty key err=%v", err)
	}
}

func TestMemorySettingsStoreEncryptsSMTPPassword(t *testing.T) {
	cipher, err := NewSettingsCipher("memory-cipher-key")
	if err != nil {
		t.Fatal(err)
	}
	settings := NewMemorySettingsStore()
	settings.SetCipher(cipher)

	if err := settings.SetMany(map[string]string{
		SettingSMTPPassword: "visible-secret",
		SettingSiteName:     "Hub",
	}); err != nil {
		t.Fatal(err)
	}

	// Raw storage must hold only the sealed form.
	if raw := settings.values[SettingSMTPPassword]; !IsEncryptedValue(raw) || strings.Contains(raw, "visible-secret") {
		t.Fatalf("raw smtp.password not encrypted: %q", raw)
	}
	if raw := settings.values[SettingSiteName]; raw != "Hub" {
		t.Fatalf("non-sensitive key must stay plaintext: %q", raw)
	}

	// Reads return the plaintext.
	got, ok, err := settings.Get(SettingSMTPPassword)
	if err != nil || !ok || got != "visible-secret" {
		t.Fatalf("Get = %q ok=%v err=%v", got, ok, err)
	}

	// Legacy plaintext rows written before encryption stay readable.
	settings.mu.Lock()
	settings.values[SettingSMTPPassword] = "old-plain"
	settings.mu.Unlock()
	got, _, _ = settings.Get(SettingSMTPPassword)
	if got != "old-plain" {
		t.Fatalf("legacy plaintext read = %q", got)
	}
}
