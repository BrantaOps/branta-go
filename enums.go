package branta

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
)

// PolygonEvmDestinationTypes are the Polygon EVM secret-key rails.
// A pol: URI maps to this family for verify (match-if-any). Token is still
// the destination type on the payment.
var PolygonEvmDestinationTypes = []DestinationType{UsdtPolygon, UsdcPolygon}

// IsPolygonEvmDestinationType reports whether type is a Polygon EVM rail.
func IsPolygonEvmDestinationType(t DestinationType) bool {
	return t == UsdtPolygon || t == UsdcPolygon
}

// IsSecretKeyAddressType reports whether type is encrypted with a random
// branta_secret (not a derived key).
func IsSecretKeyAddressType(t DestinationType) bool {
	return t == BitcoinAddress || IsPolygonEvmDestinationType(t)
}

// VerifyStatus is the result of Verify.
type VerifyStatus string

const (
	VerifyMatch      VerifyStatus = "match"
	VerifyUnknown    VerifyStatus = "unknown"
	VerifyTampered   VerifyStatus = "tampered"
	VerifyIncomplete VerifyStatus = "incomplete"
)
