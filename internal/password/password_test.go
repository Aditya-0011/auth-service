package password

import (
	"context"
	"crypto/rand"
	"testing"

	"golang.org/x/crypto/argon2"
)

func TestHashVerify(t *testing.T) {
	ctx := context.Background()
	h, err := Hash(ctx, "correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}

	match, rehash, err := Verify(ctx, "correct horse battery staple", h)
	if err != nil || !match || rehash {
		t.Fatalf("Verify(correct) = %v, %v, %v; want true, false, nil", match, rehash, err)
	}

	match, _, err = Verify(ctx, "wrong", h)
	if err != nil || match {
		t.Fatalf("Verify(wrong) = %v, %v; want false, nil", match, err)
	}
}

func TestLegacyHashNeedsRehash(t *testing.T) {
	salt := make([]byte, saltLen)
	_, _ = rand.Read(salt)
	sum := argon2.IDKey([]byte("legacy-pass"), salt, legacyIterations, legacyMemoryKiB, 1, 32)
	stored := append(salt, sum...)

	match, rehash, err := Verify(context.Background(), "legacy-pass", stored)
	if err != nil || !match || !rehash {
		t.Fatalf("Verify(legacy) = %v, %v, %v; want true, true, nil", match, rehash, err)
	}
}

func TestMalformedHash(t *testing.T) {
	if _, _, err := Verify(context.Background(), "x", []byte("$argon2id$garbage")); err == nil {
		t.Fatal("expected error for malformed hash")
	}
}
