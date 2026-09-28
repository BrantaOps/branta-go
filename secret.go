package branta

import "github.com/google/uuid"

// SecretGenerator is the secret/nonce generation strategy, injectable for testing.
type SecretGenerator interface {
	Generate() string
	DeterministicNonce() bool
}

// GuidSecretGenerator is the production SecretGenerator: random UUID v4 secrets,
// non-deterministic nonces.
type GuidSecretGenerator struct{}

func (GuidSecretGenerator) Generate() string {
	return uuid.New().String()
}

func (GuidSecretGenerator) DeterministicNonce() bool {
	return false
}
