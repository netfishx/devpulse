package auth_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethanwang/devpulse/api/internal/jwtutil"
)

const testJWTSecret = "test-secret-key"

func TestGenerateAndParseJWT(t *testing.T) {
	token, err := jwtutil.Generate(42, testJWTSecret)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	userID, err := jwtutil.Parse(token, testJWTSecret)
	require.NoError(t, err)
	assert.Equal(t, int64(42), userID)
}

func TestParseJWT_WrongSecret(t *testing.T) {
	token, err := jwtutil.Generate(42, testJWTSecret)
	require.NoError(t, err)

	_, err = jwtutil.Parse(token, "wrong-secret")
	assert.Error(t, err)
}

func TestParseJWT_InvalidToken(t *testing.T) {
	_, err := jwtutil.Parse("garbage.token.here", testJWTSecret)
	assert.Error(t, err)
}
