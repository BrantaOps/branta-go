package branta

// BrantaServerBaseURL identifies which Branta server environment to talk to.
type BrantaServerBaseURL int

const (
	Staging BrantaServerBaseURL = iota
	Production
	Localhost
)

// URL returns the base URL for this server environment (no trailing slash).
func (u BrantaServerBaseURL) URL() string {
	switch u {
	case Staging:
		return "https://staging.guardrail.branta.pro"
	case Production:
		return "https://guardrail.branta.pro"
	case Localhost:
		return "http://localhost:3000"
	default:
		return "https://staging.guardrail.branta.pro"
	}
}

// PrivacyMode controls whether plain-text (non zero-knowledge) lookups and
// destinations are permitted.
//
// Strict is the default and should be used unless a QR scanner is unavailable
// and zero-knowledge encryption is therefore impossible.
type PrivacyMode int

const (
	// PrivacyStrict forbids plain-text on-chain lookups and non-ZK destinations.
	// Never serialized over the wire.
	PrivacyStrict PrivacyMode = iota
	// PrivacyLoose removes zero-knowledge restrictions.
	PrivacyLoose
)

// DestinationType is the type of a payment destination.
//
// Wire format is snake_case and MUST match every other Branta SDK exactly for
// cross-SDK interoperability.
type DestinationType string

const (
	BitcoinAddress DestinationType = "bitcoin_address"
	Bolt11         DestinationType = "bolt11"
	Bolt12         DestinationType = "bolt12"
	LnUrl          DestinationType = "ln_url"
	TetherAddress  DestinationType = "tether_address"
	LnAddress      DestinationType = "ln_address"
	ArkAddress     DestinationType = "ark_address"
	SilentPayment  DestinationType = "silent_payment"
)
