package password

import (
	"crypto/subtle"
	"fmt"

	"golang.org/x/crypto/argon2"
)

const (
	argonTime    uint32 = 1
	argonMemory  uint32 = 32 * 1024
	argonThreads uint8  = 1
	argonKeyLen  uint32 = 32
	argonSaltLen        = 16
	argonHashLen        = argonKeyLen + argonSaltLen
)

var hashSem = make(chan struct{}, 1)

func Hash(pass []byte) ([]byte, error) {
	hashSem <- struct{}{}
	defer func() { <-hashSem }()

	salt, err := randomBytes(argonSaltLen)
	if err != nil {
		return nil, fmt.Errorf("generating salt: %w", err)
	}

	hash := argon2.IDKey(pass, salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	out := make([]byte, argonHashLen)
	copy(out[:argonSaltLen], salt)
	copy(out[argonSaltLen:], hash)

	return out, nil
}

func Compare(pass, stored []byte) (bool, error) {
	if len(stored) != int(argonHashLen) {
		return false, fmt.Errorf("invalid stored hash length: got %d, want %d", len(stored), argonHashLen)
	}

	salt := stored[:argonSaltLen]
	expectedHash := stored[argonSaltLen:]

	hashSem <- struct{}{}
	defer func() { <-hashSem }()

	candidate := argon2.IDKey(pass, salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	return subtle.ConstantTimeCompare(candidate, expectedHash) == 1, nil
}
