package branta

import "testing"

func TestBaseURLsMatchReference(t *testing.T) {
	if Staging.URL() != "https://staging.guardrail.branta.pro" {
		t.Fatalf("staging: %s", Staging.URL())
	}
	if Production.URL() != "https://guardrail.branta.pro" {
		t.Fatalf("production: %s", Production.URL())
	}
	if Localhost.URL() != "http://localhost:3000" {
		t.Fatalf("localhost: %s", Localhost.URL())
	}
}

func TestPrivacyModeDefaultsToStrict(t *testing.T) {
	if NewBrantaClientOptions(Staging).Privacy != PrivacyStrict {
		t.Fatal("expected Strict default")
	}
}

func TestDestinationTypeWireFormatIsExact(t *testing.T) {
	cases := []struct {
		variant  DestinationType
		expected string
	}{
		{BitcoinAddress, "bitcoin_address"},
		{Bolt11, "bolt11"},
		{Bolt12, "bolt12"},
		{LnUrl, "ln_url"},
		{TetherAddress, "tether_address"},
		{LnAddress, "ln_address"},
		{ArkAddress, "ark_address"},
		{SilentPayment, "silent_payment"},
	}
	for _, tc := range cases {
		if string(tc.variant) != tc.expected {
			t.Errorf("wire format mismatch for %q: got %q", tc.expected, tc.variant)
		}
	}
}
