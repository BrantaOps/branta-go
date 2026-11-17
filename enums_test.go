package branta

import (
	"strings"
	"testing"
)

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
		{UsdtBase, "usdt_base"},
		{UsdcBase, "usdc_base"},
		{UsdtArbitrum, "usdt_arbitrum"},
		{UsdcArbitrum, "usdc_arbitrum"},
	}
	if len(cases) != 18 {
		t.Fatalf("expected 18 destination types, got %d", len(cases))
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

func TestEVMTokenRailsCoverFiveNetworksWithOneTypeEach(t *testing.T) {
	chains := map[int64]bool{}
	types := map[DestinationType]bool{}
	for _, rail := range evmTokenRails {
		chains[rail.chainID] = true
		types[rail.destType] = true
		if rail.contract != strings.ToLower(rail.contract) {
			t.Errorf("contract must be lowercase: %s", rail.contract)
		}
	}
	for _, id := range []int64{1, 137, 8453, 42161, 56} {
		if !chains[id] {
			t.Errorf("missing chain %d", id)
		}
	}
	if len(chains) != 5 || len(evmTokenRails) != 10 || len(types) != 10 {
		t.Fatalf("chains=%d rails=%d types=%d", len(chains), len(evmTokenRails), len(types))
	}
}

func TestEVMTokenTypeSelectsByChainIDAndContract(t *testing.T) {
	const baseUsdc = "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913"
	if got, ok := evmTokenType(8453, baseUsdc); !ok || got != UsdcBase {
		t.Fatal(got, ok)
	}
	if got, ok := evmTokenType(8453, strings.ToLower(baseUsdc)); !ok || got != UsdcBase {
		t.Fatal(got, ok)
	}
	// Same contract on the wrong chain is not a rail.
	if _, ok := evmTokenType(1, baseUsdc); ok {
		t.Fatal("wrong chain")
	}
	// Polygon bridged USDC.e is not the native USDC contract.
	if _, ok := evmTokenType(137, "0x2791bca1f2de4661ed88a30c99a7a9449aa84174"); ok {
		t.Fatal("USDC.e")
	}
	// Arbitrum Nova is not Arbitrum One.
	if _, ok := evmTokenType(42170, "0xaf88d065e77c8cc2239327c5edb3a432268e5831"); ok {
		t.Fatal("Nova")
	}
}

func TestIsSecretKeyAddressTypeCoversBitcoinEVMRailsAndPostOnlyTypes(t *testing.T) {
	if !isSecretKeyAddressType(BitcoinAddress) {
		t.Fatal("bitcoin")
	}
	for _, rail := range evmTokenRails {
		if !isSecretKeyAddressType(rail.destType) {
			t.Fatal(rail.destType)
		}
	}
	// Tron, TON, and Solana still encrypt on AddPayment. They have no verify URI.
	for _, postOnly := range []DestinationType{UsdtTron, UsdtTon, UsdtSolana, UsdcSolana} {
		if !isSecretKeyAddressType(postOnly) {
			t.Fatal(postOnly)
		}
	}
	for _, nonSecret := range []DestinationType{Bolt11, ArkAddress, SilentPayment} {
		if isSecretKeyAddressType(nonSecret) {
			t.Fatal(nonSecret)
		}
	}
}
