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
		{UsdtPolygon, "usdt_polygon"},
		{UsdcPolygon, "usdc_polygon"},
	}
	for _, tc := range cases {
		if string(tc.variant) != tc.expected {
			t.Errorf("wire format mismatch for %q: got %q", tc.expected, tc.variant)
		}
	}
}

func TestPolygonEvmHelpers(t *testing.T) {
	if len(PolygonEvmDestinationTypes) != 2 ||
		PolygonEvmDestinationTypes[0] != UsdtPolygon ||
		PolygonEvmDestinationTypes[1] != UsdcPolygon {
		t.Fatalf("%v", PolygonEvmDestinationTypes)
	}
	if !IsPolygonEvmDestinationType(UsdtPolygon) || !IsPolygonEvmDestinationType(UsdcPolygon) {
		t.Fatal("polygon")
	}
	if IsPolygonEvmDestinationType(BitcoinAddress) {
		t.Fatal("bitcoin")
	}
	if !IsSecretKeyAddressType(BitcoinAddress) || !IsSecretKeyAddressType(UsdtPolygon) {
		t.Fatal("secret key")
	}
	if IsSecretKeyAddressType(Bolt11) {
		t.Fatal("bolt11")
	}
}
