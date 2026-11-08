package secret

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	keyPrefix         = "pk_"
	keyBytes          = 24
	keyHintLen        = 7
	codeBytes         = 10
	codeGroupLen      = 4
	RecoveryCodeCount = 10
)

var codeEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

func Hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

type APIKey struct {
	Plaintext string
	Hash      string
	Hint      string
}

func NewAPIKey() (APIKey, error) {
	b := make([]byte, keyBytes)
	if _, err := rand.Read(b); err != nil {
		return APIKey{}, fmt.Errorf("generating api key: %w", err)
	}
	key := keyPrefix + hex.EncodeToString(b)
	return APIKey{Plaintext: key, Hash: Hash(key), Hint: key[:keyHintLen]}, nil
}

func NewRecoveryCodes() (codes, hashes []string, err error) {
	codes = make([]string, RecoveryCodeCount)
	hashes = make([]string, RecoveryCodeCount)

	for i := range codes {
		b := make([]byte, codeBytes)
		if _, err := rand.Read(b); err != nil {
			return nil, nil, fmt.Errorf("generating recovery code: %w", err)
		}
		raw := codeEncoding.EncodeToString(b)
		codes[i] = group(raw)
		hashes[i] = Hash(raw)
	}
	return codes, hashes, nil
}

func HashRecoveryCode(input string) string {
	normalised := strings.Map(func(r rune) rune {
		switch {
		case r == '-' || r == ' ':
			return -1
		case r >= 'a' && r <= 'z':
			return r - 'a' + 'A'
		default:
			return r
		}
	}, input)
	return Hash(normalised)
}

func group(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && i%codeGroupLen == 0 {
			b.WriteByte('-')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func NewTemporaryPassword() (string, error) {
	b := make([]byte, codeBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generating temporary password: %w", err)
	}
	return strings.ToLower(group(codeEncoding.EncodeToString(b))), nil
}
