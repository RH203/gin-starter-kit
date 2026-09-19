package hash_test

import (
	"testing"

	"gin-starter-pack/pkg/hash"
)

func TestHashPassword_And_Check(t *testing.T) {
	password := "mySecret123!"
	hashed, err := hash.HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	if hashed == password {
		t.Errorf("hash should not equal plain password")
	}

	if !hash.CheckPasswordHash(password, hashed) {
		t.Errorf("expected password to match hash")
	}

	if hash.CheckPasswordHash("wrongPassword", hashed) {
		t.Errorf("expected wrong password to fail check")
	}
}
