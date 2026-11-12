package branta

// BrantaApiOptions is the service configuration, overridable per call.
// A nil per-call pointer leaves the service defaults in place.
// Environment zero means "unset" and does not override.
type BrantaApiOptions struct {
	Environment BrantaEnvironment
	APIKey      string
	// Deprecated: prefer a shared APIKey with PaymentBuilder.SetChildPlatform.
	HMACSecret string
}

// NewBrantaApiOptions returns options for environment with no API key or HMAC secret.
func NewBrantaApiOptions(environment BrantaEnvironment) BrantaApiOptions {
	return BrantaApiOptions{Environment: environment}
}

// GetBaseURL resolves the effective base URL.
func (o BrantaApiOptions) GetBaseURL(overrides *BrantaApiOptions) string {
	if overrides != nil && overrides.Environment != envUnset {
		return overrides.Environment.URL()
	}
	return o.Environment.URL()
}

// GetAPIKey resolves the effective API key. An override wins when non-empty.
func (o BrantaApiOptions) GetAPIKey(overrides *BrantaApiOptions) string {
	if overrides != nil && overrides.APIKey != "" {
		return overrides.APIKey
	}
	return o.APIKey
}

// GetHMACSecret resolves the effective HMAC secret. An override wins when non-empty.
func (o BrantaApiOptions) GetHMACSecret(overrides *BrantaApiOptions) string {
	if overrides != nil && overrides.HMACSecret != "" {
		return overrides.HMACSecret
	}
	return o.HMACSecret
}
