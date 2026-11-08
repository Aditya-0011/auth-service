package password

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/crypto/argon2"
)

const (
	memoryKiB   uint32 = 19 * 1024
	iterations  uint32 = 2
	parallelism uint8  = 1
	keyLen      uint32 = 32
	saltLen            = 16
)

const (
	legacyMemoryKiB  uint32 = 32 * 1024
	legacyIterations uint32 = 1
	legacyLen               = saltLen + 32
)

var errMalformedHash = errors.New("malformed password hash")

var b64 = base64.RawStdEncoding

var sem = make(chan struct{}, 1)

func acquire(ctx context.Context) error {
	select {
	case sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func release() { <-sem }

func Hash(ctx context.Context, pass string) ([]byte, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generating salt: %w", err)
	}

	if err := acquire(ctx); err != nil {
		return nil, err
	}
	sum := argon2.IDKey([]byte(pass), salt, iterations, memoryKiB, parallelism, keyLen)
	release()

	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, memoryKiB, iterations, parallelism, b64.EncodeToString(salt), b64.EncodeToString(sum))
	return []byte(encoded), nil
}

func Verify(ctx context.Context, pass string, stored []byte) (match, needsRehash bool, err error) {
	var (
		salt, want []byte
		mem, iter  uint32
		lanes      uint8
	)

	if len(stored) == legacyLen && !strings.HasPrefix(string(stored), "$") {
		salt, want = stored[:saltLen], stored[saltLen:]
		mem, iter, lanes = legacyMemoryKiB, legacyIterations, 1
		needsRehash = true
	} else {
		salt, want, mem, iter, lanes, err = decode(string(stored))
		if err != nil {
			return false, false, err
		}
		needsRehash = mem != memoryKiB || iter != iterations || lanes != parallelism || len(want) != int(keyLen)
	}

	if err := acquire(ctx); err != nil {
		return false, false, err
	}
	got := argon2.IDKey([]byte(pass), salt, iter, mem, lanes, uint32(len(want)))
	release()

	if subtle.ConstantTimeCompare(got, want) != 1 {
		return false, false, nil
	}
	return true, needsRehash, nil
}

var dummyHash = sync.OnceValues(func() ([]byte, error) {
	return Hash(context.Background(), "dummy-password-for-timing")
})

func BurnTime(ctx context.Context) {
	if h, err := dummyHash(); err == nil {
		_, _, _ = Verify(ctx, "not-the-password", h)
	}
}

func decode(s string) (salt, sum []byte, mem, iter uint32, lanes uint8, err error) {
	parts := strings.Split(s, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return nil, nil, 0, 0, 0, errMalformedHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return nil, nil, 0, 0, 0, errMalformedHash
	}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &mem, &iter, &lanes); err != nil || mem == 0 || iter == 0 || lanes == 0 {
		return nil, nil, 0, 0, 0, errMalformedHash
	}
	if salt, err = b64.DecodeString(parts[4]); err != nil {
		return nil, nil, 0, 0, 0, errMalformedHash
	}
	if sum, err = b64.DecodeString(parts[5]); err != nil || len(sum) == 0 {
		return nil, nil, 0, 0, 0, errMalformedHash
	}
	return salt, sum, mem, iter, lanes, nil
}
