package auth_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/t0mer/aghsync/internal/auth"
)

func TestHashPassword_AndCheck(t *testing.T) {
	hash, err := auth.HashPassword("correct-horse-battery-staple")
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.True(t, auth.CheckPassword("correct-horse-battery-staple", hash))
	assert.False(t, auth.CheckPassword("wrong-password", hash))
}

func TestHashPassword_EmptyInput(t *testing.T) {
	hash, err := auth.HashPassword("")
	require.NoError(t, err) // bcrypt accepts empty strings
	assert.True(t, auth.CheckPassword("", hash))
	assert.False(t, auth.CheckPassword("x", hash))
}

func TestGenerateToken_UniqueAndHashable(t *testing.T) {
	plain1, hash1, err := auth.GenerateToken()
	require.NoError(t, err)
	assert.Len(t, plain1, 64) // 32 bytes → 64 hex chars
	assert.NotEmpty(t, hash1)

	plain2, hash2, err := auth.GenerateToken()
	require.NoError(t, err)
	assert.NotEqual(t, plain1, plain2)
	assert.NotEqual(t, hash1, hash2)

	assert.True(t, auth.CheckToken(plain1, hash1))
	assert.False(t, auth.CheckToken(plain1, hash2))
	assert.True(t, auth.CheckToken(plain2, hash2))
}

func TestEncryptDecryptPassword(t *testing.T) {
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = byte(i + 1) // deterministic for test
	}

	enc, err := auth.EncryptPassword("super-secret-agh-password", secret)
	require.NoError(t, err)
	assert.NotEmpty(t, enc)
	assert.NotEqual(t, "super-secret-agh-password", enc)

	dec, err := auth.DecryptPassword(enc, secret)
	require.NoError(t, err)
	assert.Equal(t, "super-secret-agh-password", dec)
}

func TestEncryptPassword_NonDeterministic(t *testing.T) {
	// Each call produces a different ciphertext (random nonce).
	secret := make([]byte, 32)
	enc1, err := auth.EncryptPassword("password", secret)
	require.NoError(t, err)
	enc2, err := auth.EncryptPassword("password", secret)
	require.NoError(t, err)
	assert.NotEqual(t, enc1, enc2)
}

func TestDecryptPassword_WrongSecret(t *testing.T) {
	secret := make([]byte, 32)
	enc, err := auth.EncryptPassword("password", secret)
	require.NoError(t, err)

	wrongSecret := make([]byte, 32)
	wrongSecret[0] = 0xFF
	_, err = auth.DecryptPassword(enc, wrongSecret)
	assert.Error(t, err)
}

func TestDecryptPassword_Corrupted(t *testing.T) {
	secret := make([]byte, 32)
	_, err := auth.DecryptPassword("not-valid-base64!!!", secret)
	assert.Error(t, err)
}
