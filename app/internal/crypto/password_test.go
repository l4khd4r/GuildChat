package crypto

import "testing"

func TestPasswordHash(t *testing.T) {
	password := "secret123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("hash: %s", hash)

	valid, err := VerifyPassword(password, hash)
	if err != nil {
		t.Fatal(err)
	}

	if !valid {
		t.Fatal("password should be valid")
	}

	invalid, err := VerifyPassword("wrong-password", hash)
	if err != nil {
		t.Fatal(err)
	}

	if invalid {
		t.Fatal("wrong password should be rejected")
	}
}
