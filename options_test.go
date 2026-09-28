package branta

import "testing"

func TestOverrideWinsOverDefaultFieldByField(t *testing.T) {
	defaultOpts := BrantaClientOptions{
		BaseURL:       Staging,
		DefaultAPIKey: "default-key",
		HMACSecret:    "default-hmac",
		Privacy:       PrivacyStrict,
	}
	over := NewBrantaClientOptions(Production)
	over.DefaultAPIKey = "child-key"
	// HMACSecret left empty -> should fall back to default's hmac secret.

	if got := defaultOpts.GetBaseURL(&over); got != "https://guardrail.branta.pro" {
		t.Fatalf("base url: %s", got)
	}
	if got := defaultOpts.GetAPIKey(&over); got != "child-key" {
		t.Fatalf("api key: %s", got)
	}
	if got := defaultOpts.GetHMACSecret(&over); got != "default-hmac" {
		t.Fatalf("hmac: %s", got)
	}
}

func TestNilOverridesFallBackEntirelyToDefault(t *testing.T) {
	defaultOpts := NewBrantaClientOptions(Staging)
	if got := defaultOpts.GetBaseURL(nil); got != "https://staging.guardrail.branta.pro" {
		t.Fatalf("base url: %s", got)
	}
	if defaultOpts.GetPrivacy(nil) != PrivacyStrict {
		t.Fatal("privacy")
	}
	if defaultOpts.GetAPIKey(nil) != "" {
		t.Fatal("api key")
	}
	if defaultOpts.GetHMACSecret(nil) != "" {
		t.Fatal("hmac")
	}
}
