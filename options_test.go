package branta

import "testing"

func TestOverrideWinsOverDefaultFieldByField(t *testing.T) {
	defaultOpts := BrantaApiOptions{
		Environment: Staging,
		APIKey:      "default-key",
		HMACSecret:  "default-hmac",
	}
	over := NewBrantaApiOptions(Production)
	over.APIKey = "child-key"

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

func TestZeroEnvironmentDoesNotOverride(t *testing.T) {
	defaultOpts := NewBrantaApiOptions(Production)
	over := &BrantaApiOptions{APIKey: "child-key"}
	if got := defaultOpts.GetBaseURL(over); got != "https://guardrail.branta.pro" {
		t.Fatalf("base url: %s", got)
	}
}

func TestNilOverridesFallBackEntirelyToDefault(t *testing.T) {
	defaultOpts := NewBrantaApiOptions(Staging)
	if got := defaultOpts.GetBaseURL(nil); got != "https://staging.guardrail.branta.pro" {
		t.Fatalf("base url: %s", got)
	}
	if defaultOpts.GetAPIKey(nil) != "" {
		t.Fatal("api key")
	}
	if defaultOpts.GetHMACSecret(nil) != "" {
		t.Fatal("hmac")
	}
}
