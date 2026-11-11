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
		{UsdcTon, "usdc_ton"},
		{UsdtSolana, "usdt_solana"},
		{UsdcSolana, "usdc_solana"},
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

func TestSecretKeyURIFamilies(t *testing.T) {
	want := map[string][]DestinationType{
		"pol":  {UsdtPolygon, UsdcPolygon},
		"eth":  {UsdtEthereum, UsdcEthereum},
		"bsc":  {UsdtBsc, UsdcBsc},
		"tron": {UsdtTron},
		"ton":  {UsdtTon, UsdcTon},
		"sol":  {UsdtSolana, UsdcSolana},
	}
	if len(SecretKeyURIFamilies) != len(want) {
		t.Fatalf("%v", SecretKeyURIFamilies)
	}
	for scheme, family := range want {
		got := SecretKeyURIFamilies[scheme]
		if len(got) != len(family) {
			t.Fatalf("%s: %v", scheme, got)
		}
		for i := range family {
			if got[i] != family[i] {
				t.Fatalf("%s: %v", scheme, got)
			}
			if !IsSecretKeyAddressType(family[i]) {
				t.Fatalf("%s not secret key", family[i])
			}
		}
	}
	if IsPolygonEvmDestinationType(UsdtEthereum) {
		t.Fatal("ethereum is not polygon")
	}
	for _, nonSecret := range []DestinationType{Bolt11, ArkAddress, SilentPayment} {
		if IsSecretKeyAddressType(nonSecret) {
			t.Fatal(nonSecret)
		}
	}
}
