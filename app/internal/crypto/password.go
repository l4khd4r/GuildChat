package crypto

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	saltLength = 16
	argonTime    = 1
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeyLen  = 32
)

func HashPassword(password string) (string, error) {
	salt := make([]byte, saltLength)

	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		argonTime,
		argonMemory,
		argonThreads,
		argonKeyLen,
	)

	encodedSalt := base64.RawStdEncoding.EncodeToString(salt)
	encodedHash := base64.RawStdEncoding.EncodeToString(hash)

	return fmt.Sprintf(
		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argonMemory,
		argonTime,
		argonThreads,
		encodedSalt,
		encodedHash,
	), nil
}

func VerifyPassword(password, encodedHash string) (bool, error) {
	parts := strings.Split(encodedHash, "$")

	if len(parts) != 6 {
		return false, errors.New("invalid password hash format")
	}

	if parts[1] != "argon2id" {
		return false, errors.New("invalid password hash algorithm")
	}

	// parts[2] = v=19
	// parts[3] = m=65536,t=1,p=4
	// parts[4] = salt
	// parts[5] = hash

	var memory uint32
	var time uint32
	var threads uint8

	_, err := fmt.Sscanf(
		parts[3],
		"m=%d,t=%d,p=%d",
		&memory,
		&time,
		&threads,
	)
	if err != nil {
		return false, errors.New("invalid Argon2 parameters")
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, errors.New("invalid salt")
	}

	expectedHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, errors.New("invalid password hash")
	}

	actualHash := argon2.IDKey(
		[]byte(password),
		salt,
		time,
		memory,
		threads,
		uint32(len(expectedHash)),
	)

	if subtle.ConstantTimeCompare(actualHash, expectedHash) == 1 {
		return true, nil
	}

	return false, nil
}
// func VerifyPassword(password, encodedHash string) (bool, error) {
// 	var memory uint32
// 	var time uint32
// 	var threads uint8
// 	var encodedSalt string
// 	var encodedPasswordHash string

// 	_, err := fmt.Sscanf(
// 		encodedHash,
// 		"$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
// 		&memory,
// 		&time,
// 		&threads,
// 		&encodedSalt,
// 		&encodedPasswordHash,
// 	)

// 	if err != nil {
// 		return false, errors.New("invalid password hash format")
// 	}

// 	salt, err := base64.RawStdEncoding.DecodeString(encodedSalt)
// 	if err != nil {
// 		return false, errors.New("invalid salt")
// 	}

// 	expectedHash, err := base64.RawStdEncoding.DecodeString(encodedPasswordHash)
// 	if err != nil {
// 		return false, errors.New("invalid password hash")
// 	}

// 	actualHash := argon2.IDKey(
// 		[]byte(password),
// 		salt,
// 		time,
// 		memory,
// 		threads,
// 		uint32(len(expectedHash)),
// 	)

// 	if string(actualHash) == string(expectedHash) {
// 		return true, nil
// 	}

// 	return false, nil
// }
