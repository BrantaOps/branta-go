package branta

// BrantaClientOptions is per-service configuration, overridable per-call.
type BrantaClientOptions struct {
	BaseURL       BrantaServerBaseURL
	DefaultAPIKey string
	HMACSecret    string
	Privacy       PrivacyMode
}

// NewBrantaClientOptions returns options for baseURL with PrivacyStrict and
// no API key or HMAC secret.
func NewBrantaClientOptions(baseURL BrantaServerBaseURL) BrantaClientOptions {
	return BrantaClientOptions{
		BaseURL: baseURL,
		Privacy: PrivacyStrict,
	}
}

// GetBaseURL resolves the effective base URL: overrides wins if present, else o.
func (o BrantaClientOptions) GetBaseURL(overrides *BrantaClientOptions) string {
	if overrides != nil {
		return overrides.BaseURL.URL()
	}
	return o.BaseURL.URL()
}

// GetPrivacy resolves the effective privacy mode: overrides wins if present, else o.
func (o BrantaClientOptions) GetPrivacy(overrides *BrantaClientOptions) PrivacyMode {
	if overrides != nil {
		return overrides.Privacy
	}
	return o.Privacy
}

// GetAPIKey resolves the effective API key, field-by-field: overrides' key
// wins if non-empty, else falls back to o's key. Empty string means unset.
func (o BrantaClientOptions) GetAPIKey(overrides *BrantaClientOptions) string {
	if overrides != nil && overrides.DefaultAPIKey != "" {
		return overrides.DefaultAPIKey
	}
	return o.DefaultAPIKey
}

// GetHMACSecret resolves the effective HMAC secret, field-by-field: overrides'
// secret wins if non-empty, else falls back to o's secret.
func (o BrantaClientOptions) GetHMACSecret(overrides *BrantaClientOptions) string {
	if overrides != nil && overrides.HMACSecret != "" {
		return overrides.HMACSecret
	}
	return o.HMACSecret
}
