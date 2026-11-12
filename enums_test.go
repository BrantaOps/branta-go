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
		{UsdtEthereum, "usdt_ethereum"},
		{UsdcEthereum, "usdc_ethereum"},
		{UsdtBsc, "usdt_bsc"},
		{UsdcBsc, "usdc_bsc"},
		{UsdtTron, "usdt_tron"},
		{UsdtTon, "usdt_ton"},
		{UsdtSolana, "usdt_solana"},
		{UsdcSolana, "usdc_solana"},
	}
	if len(cases) != 14 {
		t.Fatalf("expected 14 destination types, got %d", len(cases))
	}
	for _, tc := range cases {
		if string(tc.variant) != tc.expected {
			t.Errorf("wire format mismatch for %q: got %q", tc.expected, tc.variant)
		}
	}
	if DestinationType("usdc_ton") == UsdcSolana || DestinationType("usdc_ton") == UsdtTon {
		t.Fatal("usdc_ton must stay deferred")
	}
}

func TestSecretKeyURIFamilies(t *testing.T) {
	want := map[string][]DestinationType{
		"pol":  {UsdtPolygon, UsdcPolygon},
		"eth":  {UsdtEthereum, UsdcEthereum},
		"bsc":  {UsdtBsc, UsdcBsc},
		"tron": {UsdtTron},
		"ton":  {UsdtTon},
		"sol":  {UsdtSolana, UsdcSolana},
	}
	if len(secretKeyURIFamilies) != len(want) {
		t.Fatalf("%v", secretKeyURIFamilies)
	}
	for scheme, family := range want {
		got := secretKeyURIFamilies[scheme]
		if len(got) != len(family) {
			t.Fatalf("%s: %v", scheme, got)
		}
		for i := range family {
			if got[i] != family[i] {
				t.Fatalf("%s: %v", scheme, got)
			}
			if !isSecretKeyAddressType(family[i]) {
				t.Fatalf("%s not secret key", family[i])
			}
		}
	}
	if !isSecretKeyAddressType(BitcoinAddress) {
		t.Fatal("bitcoin")
	}
	for _, nonSecret := range []DestinationType{Bolt11, ArkAddress, SilentPayment} {
		if isSecretKeyAddressType(nonSecret) {
			t.Fatal(nonSecret)
		}
	}
}
