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

func TestUnsetEnvironmentUsesStaging(t *testing.T) {
	if BrantaEnvironment(0).URL() != "https://staging.guardrail.branta.pro" {
		t.Fatal("zero value")
	}
}

func TestDestinationTypeWireFormatIsExact(t *testing.T) {
	cases := []struct {
		variant  DestinationType
		expected string
	}{
		{BitcoinAddress, "bitcoin_address"},
		{Bolt11, "bolt11"},
		{ArkAddress, "ark_address"},
		{SilentPayment, "silent_payment"},
	}
	for _, tc := range cases {
		if string(tc.variant) != tc.expected {
			t.Errorf("wire format mismatch for %q: got %q", tc.expected, tc.variant)
		}
	}
}
