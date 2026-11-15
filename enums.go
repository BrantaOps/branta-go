package branta

import "strings"

// BrantaEnvironment identifies which Branta server to talk to.
// The zero value means unset; URL() then uses staging.
type BrantaEnvironment int

const (
	envUnset BrantaEnvironment = iota
	Staging
	Production
	Localhost
)

// URL returns the base URL for this environment (no trailing slash).
func (e BrantaEnvironment) URL() string {
	switch e {
	case Production:
		return "https://guardrail.branta.pro"
	case Localhost:
		return "http://localhost:3000"
	default:
		return "https://staging.guardrail.branta.pro"
	}
}

// DestinationType is a destination AddPayment can encrypt.
// Wire format is snake_case and must match every other Branta SDK.
type DestinationType string

const (
	BitcoinAddress DestinationType = "bitcoin_address"
	Bolt11         DestinationType = "bolt11"
	ArkAddress     DestinationType = "ark_address"
	SilentPayment  DestinationType = "silent_payment"
	UsdtPolygon    DestinationType = "usdt_polygon"
	UsdcPolygon    DestinationType = "usdc_polygon"
	UsdtEthereum   DestinationType = "usdt_ethereum"
	UsdcEthereum   DestinationType = "usdc_ethereum"
	UsdtBsc        DestinationType = "usdt_bsc"
	UsdcBsc        DestinationType = "usdc_bsc"
	UsdtTron       DestinationType = "usdt_tron"
	UsdtTon        DestinationType = "usdt_ton"
	UsdtSolana     DestinationType = "usdt_solana"
	UsdcSolana     DestinationType = "usdc_solana"
	UsdtBase       DestinationType = "usdt_base"
	UsdcBase       DestinationType = "usdc_base"
	UsdtArbitrum   DestinationType = "usdt_arbitrum"
	UsdcArbitrum   DestinationType = "usdc_arbitrum"
)

// evmTokenRail is an ERC-681 `ethereum:<contract>@<chainId>/transfer` rail.
// Unexported — consumers use DestinationType constants and Verify / AddPayment only.
type evmTokenRail struct {
	chainID  int64
	contract string // lowercase, so lookup is case-insensitive
	destType DestinationType
}

// evmTokenRails maps chain id plus token contract to one destination type.
var evmTokenRails = []evmTokenRail{
	{1, "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", UsdcEthereum},
	{1, "0xdac17f958d2ee523a2206206994597c13d831ec7", UsdtEthereum},
	{137, "0x3c499c542cef5e3811e1192ce70d8cc03d5c3359", UsdcPolygon},
	{137, "0xc2132d05d31c914a87c6611c10748aeb04b58e8f", UsdtPolygon},
	{8453, "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913", UsdcBase},
	{8453, "0xfde4c96c8593536e31f229ea8f37b2ada2699bb2", UsdtBase},
	{42161, "0xaf88d065e77c8cc2239327c5edb3a432268e5831", UsdcArbitrum},
	{42161, "0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9", UsdtArbitrum},
	{56, "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d", UsdcBsc},
	{56, "0x55d398326f99059ff775485246999027b3197955", UsdtBsc},
}

// evmTokenType returns the destination type for a chain id and token contract.
func evmTokenType(chainID int64, contract string) (DestinationType, bool) {
	needle := strings.ToLower(contract)
	for _, rail := range evmTokenRails {
		if rail.chainID == chainID && rail.contract == needle {
			return rail.destType, true
		}
	}
	return "", false
}

// uuidSecretDestinationTypes are the address rails AddPayment encrypts with one
// UUID secret. Tron, TON, and Solana are included because AddPayment still
// encrypts them. They have no verify URI.
var uuidSecretDestinationTypes = func() map[DestinationType]struct{} {
	set := map[DestinationType]struct{}{
		BitcoinAddress: {},
		UsdtTron:       {},
		UsdtTon:        {},
		UsdtSolana:     {},
		UsdcSolana:     {},
	}
	for _, rail := range evmTokenRails {
		set[rail.destType] = struct{}{}
	}
	return set
}()

// isSecretKeyAddressType reports whether type is encrypted with a random
// branta_secret (not a derived key).
func isSecretKeyAddressType(t DestinationType) bool {
	_, ok := uuidSecretDestinationTypes[t]
	return ok
}

// VerifyStatus is the result of Verify.
type VerifyStatus string

const (
	VerifyMatch      VerifyStatus = "match"
	VerifyUnknown    VerifyStatus = "unknown"
	VerifyTampered   VerifyStatus = "tampered"
	VerifyIncomplete VerifyStatus = "incomplete"
)
