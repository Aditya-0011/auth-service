package secret

import (
	"strings"
	"testing"
)

func TestAPIKey(t *testing.T) {
	k, err := NewAPIKey()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(k.Plaintext, "pk_") || len(k.Plaintext) != 51 {
		t.Fatalf("unexpected key format %q", k.Plaintext)
	}
	if k.Hash != Hash(k.Plaintext) || k.Hint != k.Plaintext[:7] {
		t.Fatal("hash or hint does not match plaintext")
	}
}

func TestRecoveryCodesNormalise(t *testing.T) {
	codes, hashes, err := NewRecoveryCodes()
	if err != nil {
		t.Fatal(err)
	}
	if len(codes) != RecoveryCodeCount {
		t.Fatalf("got %d codes", len(codes))
	}
	seen := map[string]bool{}
	for i, c := range codes {
		if len(c) != 19 || strings.Count(c, "-") != 3 {
			t.Fatalf("unexpected code format %q", c)
		}
		messy := " " + strings.ToLower(strings.ReplaceAll(c, "-", " ")) + " "
		if HashRecoveryCode(messy) != hashes[i] || HashRecoveryCode(c) != hashes[i] {
			t.Fatalf("normalised hash mismatch for %q", c)
		}
		if seen[c] {
			t.Fatal("duplicate code")
		}
		seen[c] = true
	}
}
