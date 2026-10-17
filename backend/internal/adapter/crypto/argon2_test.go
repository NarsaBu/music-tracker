package crypto

import (
	"errors"
	"testing"
)

// Cheap parameters keep the test fast; the format is the same.
var testParams = Argon2Params{Memory: 1024, Time: 1, Threads: 1, SaltLen: 16, KeyLen: 32}

func TestArgon2Hasher_RoundTrip(t *testing.T) {
	h := NewArgon2Hasher(testParams)

	encoded, err := h.Hash("correct horse")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	tests := []struct {
		password string
		want     bool
	}{
		{"correct horse", true},
		{"wrong horse", false},
		{"", false},
	}
	for _, tt := range tests {
		ok, err := h.Verify(tt.password, encoded)
		if err != nil {
			t.Fatalf("Verify(%q) error = %v", tt.password, err)
		}
		if ok != tt.want {
			t.Errorf("Verify(%q) = %v, want %v", tt.password, ok, tt.want)
		}
	}
}

func TestArgon2Hasher_SaltIsRandom(t *testing.T) {
	h := NewArgon2Hasher(testParams)
	a, _ := h.Hash("same")
	b, _ := h.Hash("same")
	if a == b {
		t.Error("two hashes of the same password are equal, salt is not random")
	}
}

func TestArgon2Hasher_InvalidHash(t *testing.T) {
	h := NewArgon2Hasher(testParams)
	for _, encoded := range []string{"", "plain", "$bcrypt$v=19$m=1,t=1,p=1$c2FsdA$a2V5", "$argon2id$v=19$m=x$salt$key"} {
		if _, err := h.Verify("x", encoded); !errors.Is(err, ErrInvalidHash) {
			t.Errorf("Verify(_, %q) error = %v, want ErrInvalidHash", encoded, err)
		}
	}
}
